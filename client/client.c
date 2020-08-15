/*
 * This file is part of yodi.
 *
 * Copyright 2020 Dima Krasner
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <signal.h>
#include <stdint.h>
#include <unistd.h>
#include <stdlib.h>
#include <signal.h>
#include <errno.h>

#include <MQTTClient.h>
#include <boydemdb.h>

#include <yodi.h>

#define MQTT_BUFSIZ 1024 * 1024
#define MQTT_TIMEOUT 3000

#define CONNECT_INTERVAL 1
#define CONNECT_TRIES 5
#define CONNECT_TIMEOUT 3000

#define SIGMQTT SIGRTMIN
#define RESULT_POLL_INTERVAL 1

#ifdef YODI_SSL
#	define MQTT_PORT 8883
#else
#	define MQTT_PORT 1883
#endif

static void on_unknown_message(MessageData* md)
{
	yodi_warn("Unexpected message from %.*s",
	          md->topicName->lenstring.len,
	          md->topicName->lenstring.data);
}

static void on_set(MessageData* md)
{
	MQTTMessage* message = md->message;
	boydemdb db = md->private;

	yodi_debug("Received command %.*s",
	           (int)(message->payloadlen % INT_MAX),
	           (char *)message->payload);

	boydemdb_add(db,
	             YODI_TYPE_COMMAND,
	             message->payload,
	             message->payloadlen);
}

static int publish_items(MQTTClient *c,
                         const boydemdb_type type,
                         const char *topic,
                         const enum QoS qos,
                         boydemdb db,
                         const int log)
{
	MQTTMessage msg = {.qos = qos};
	boydemdb_id id;
	size_t len;
	yodi_autofree void *buf = NULL;
	int ret;

	while (1) {
		buf = boydemdb_one(db, type, &id, &len);
		if (!buf)
			break;

		if (log)
			yodi_debug("Publishing %.*s to %s",
			           (int)(len % INT_MAX),
			           (char *)buf,
			           topic);

		msg.payload = buf;
		msg.payloadlen = len;

		ret = MQTTPublish(c, topic, &msg);
		if (ret != SUCCESS)
			return ret;

		boydemdb_delete(db, id);
	}

	return SUCCESS;
}

static int publish_results(MQTTClient *c,
                           const char *topic,
                           boydemdb db)
{
	return publish_items(c, YODI_TYPE_RESULT, topic, QOS1, db, 1);
}

static int publish_logs(MQTTClient *c,
                        const char *topic,
                        boydemdb db)
{
	return publish_items(c, YODI_TYPE_LOG, topic, QOS0, db, 0);
}

#ifdef YODI_HAVE_KRISA

static int report_crashes(MQTTClient *c,
                          const char *topic,
                          boydemdb db)
{
	return publish_items(c, YODI_TYPE_BACKTRACE, topic, QOS1, db, 1);
}

#endif

int yodi_client(int argc, char *argv[])
{
	static char cmd_topic[256], result_topic[256], log_topic[256];
#ifdef YODI_HAVE_KRISA
	static char backtrace_topic[256];
#endif
	MQTTPacket_connectData data = MQTTPacket_connectData_initializer;
	Network n;
	MQTTClient c;
	struct timespec ts = {.tv_sec = CONNECT_INTERVAL};
	siginfo_t si;
	sigset_t set;
	yodi_db_autoclose boydemdb db = BOYDEMDB_INIT;
	yodi_autofree unsigned char *rxbuf = NULL, *txbuf = NULL;
	char *host = NULL, *uri = NULL, *id = NULL, *user = NULL, *password = NULL;
	long tmp;
	int port = MQTT_PORT, ret = EXIT_FAILURE;
	unsigned int i;

	while (1) {
		switch (getopt(argc, argv, "h:u:p:i:U:P:")) {
		case '?':
			return EXIT_FAILURE;

		case -1:
			goto parsed;

		case 'h':
			host = optarg;
			break;

		case 'u':
			uri = optarg;
			break;

		case 'p':
			tmp = strtol(optarg, NULL, 10);
			if ((tmp <= 0) || (tmp > UINT16_MAX))
				return EXIT_FAILURE;
			port = (int)tmp;
			break;

		case 'i':
			id = optarg;
			break;

		case 'U':
			user = optarg;
			break;

		case 'P':
			password = optarg;
			break;
		}
	}

parsed:
	if (!host || !uri || !id || !user || !password)
		return EXIT_FAILURE;

	if ((sigemptyset(&set) < 0) ||
	    (sigaddset(&set, SIGTERM) < 0) ||
	    (sigaddset(&set, SIGMQTT) < 0) ||
	    (sigprocmask(SIG_BLOCK, &set, NULL) < 0))
		return EXIT_FAILURE;

	rxbuf = malloc(MQTT_BUFSIZ);
	if (!rxbuf)
		return EXIT_FAILURE;

	txbuf = malloc(MQTT_BUFSIZ);
	if (!txbuf)
		return EXIT_FAILURE;

	db = boydemdb_open(YODI_DB_PATH);
	if (!db)
		return EXIT_FAILURE;

	NetworkInit(&n);

	for (i = 0; i < CONNECT_TRIES; ++i) {
		yodi_debug("Connecting to %s:%d%s", host, port, uri);

		if (NetworkConnectURI(&n, host, port, uri, CONNECT_TIMEOUT) == SUCCESS)
			goto connected;

		if (sigtimedwait(&set, &si, &ts) < 0) {
			if (errno == EAGAIN)
				continue;
		}

		return EXIT_FAILURE;
	}

	return EXIT_FAILURE;

connected:
	if (yodi_setsig(n.my_socket, SIGMQTT) < 0) {
		NetworkDisconnect(&n);
		return EXIT_FAILURE;
	}

	MQTTClientInit(&c,
	               &n,
	               MQTT_TIMEOUT,
	               rxbuf,
	               MQTT_BUFSIZ,
	               txbuf,
	               MQTT_BUFSIZ);

	c.defaultMessageHandler = on_unknown_message;

	data.willFlag = 0;
	data.MQTTVersion = 4;
	data.clientID.cstring = id;
	data.username.cstring = user;
	data.password.cstring = password;

	data.keepAliveInterval = 20;
	data.cleansession = 1;

	if (MQTTConnect(&c, &data) != SUCCESS) {
		NetworkDisconnect(&n);
		return EXIT_FAILURE;
	}

	snprintf(cmd_topic, sizeof(cmd_topic), "/%s/commands", id);
	snprintf(result_topic, sizeof(result_topic), "/%s/results", id);
	snprintf(log_topic, sizeof(log_topic), "/%s/log", id);
#ifdef YODI_HAVE_KRISA
	snprintf(backtrace_topic, sizeof(backtrace_topic), "/%s/crashes", id);
#endif

	yodi_debug("Subscribing to %s", cmd_topic);
	if (MQTTSubscribe(&c, cmd_topic, QOS1, on_set) != SUCCESS)
		goto cleanup;
	yodi_debug("Subscribed to %s", cmd_topic);

	c.private = db;
	ts.tv_sec = RESULT_POLL_INTERVAL;

	while (1) {
		if (sigtimedwait(&set, &si, &ts) < 0) {
			if (errno != EAGAIN)
				break;
		}
		else {
			if (si.si_signo != SIGMQTT) {
				ret = EXIT_SUCCESS;
				break;
			}

			if (MQTTYield(&c, MQTT_TIMEOUT) != SUCCESS)
				break;
		}

		if (publish_results(&c, result_topic, db) != SUCCESS)
			break;

		if (publish_logs(&c, log_topic, db) != SUCCESS)
			break;

#ifdef YODI_HAVE_KRISA
		if (report_crashes(&c, backtrace_topic, db) != SUCCESS)
			break;
#endif
	}

	yodi_debug("Unsubscribing from %s", cmd_topic);
	MQTTUnsubscribe(&c, cmd_topic);
	ret = EXIT_SUCCESS;

cleanup:
	yodi_debug("Disconnecting from %s:%d%s", host, port, uri);
	MQTTDisconnect(&c);
	NetworkDisconnect(&n);

	return ret;
}
