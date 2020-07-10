#include <unistd.h>
#include <stdlib.h>
#include <signal.h>
#include <errno.h>

#include <MQTTClient.h>
#include <boydemdb.h>
#include <yodi.h>

#define MQTT_BUFSIZ 1024 * 1024
#define MQTT_TIMEOUT 5000

#define SIGMQTT 5
#define RESULT_POLL_INTERVAL 5

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

	boydemdb_set(db,
	             YODI_TYPE_COMMAND,
	             message->payload,
	             message->payloadlen);
}

static int publish_results(MQTTClient *c,
                           const char *topic,
                           boydemdb db)
{
	MQTTMessage msg = {.qos = QOS0};
	boydemdb_id id;
	size_t len;
	yodi_autofree void *buf = NULL;
	int ret;

	while (1) {
		buf = boydemdb_one(db, YODI_TYPE_RESULT, &id, &len);
		if (!buf)
			break;

		yodi_debug("Publishing result: %.*s",
		           (int)(len % INT_MAX),
		           (char *)buf);

		msg.payload = buf;
		msg.payloadlen = len;

		ret = MQTTPublish(c, topic, &msg);
		if (ret != SUCCESS)
			return ret;

		boydemdb_delete(db, id);
	}

	return SUCCESS;
}

int yodi_client(int argc, char *argv[])
{
	static char cmd_topic[256], result_topic[256];
	MQTTPacket_connectData data = MQTTPacket_connectData_initializer;
	Network n;
	MQTTClient c;
	struct timespec ts = {.tv_sec = RESULT_POLL_INTERVAL};
	siginfo_t si;
	sigset_t set;
	yodi_db_autoclose boydemdb db = BOYDEMDB_INIT;
	yodi_autofree unsigned char *rxbuf = NULL, *txbuf = NULL;
	char *host = NULL, *id = NULL, *user = NULL, *password = NULL;
	int ret = EXIT_FAILURE;

	while (1) {
		switch (getopt(argc, argv, "h:p:i:u:p:")) {
		case '?':
			return EXIT_FAILURE;

		case -1:
			goto parsed;

		case 'h':
			host = optarg;
			break;

		case 'i':
			id = optarg;
			break;

		case 'u':
			user = optarg;
			break;

		case 'p':
			password = optarg;
			break;
		}
	}

parsed:
	if (!host || !id || !user || !password)
		return EXIT_FAILURE;

	if ((sigemptyset(&set) < 0) ||
	    (sigaddset(&set, SIGTERM) < 0) ||
	    (sigaddset(&set, SIGMQTT) < 0))
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
	yodi_debug("Connecting to %s:%hu", host, MQTT_PORT);
	if (NetworkConnect(&n, host, MQTT_PORT) != SUCCESS)
		return EXIT_FAILURE;

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

	yodi_debug("Subscribing to %s", cmd_topic);
	if (MQTTSubscribe(&c, cmd_topic, 1, on_set) != SUCCESS)
		goto cleanup;
	yodi_debug("Subscribed to %s", cmd_topic);

	c.private = db;

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
	}

	yodi_debug("Unsubscribing from %s", cmd_topic);
	MQTTUnsubscribe(&c, cmd_topic);
	ret = EXIT_SUCCESS;

cleanup:
	yodi_debug("Disconnecting from %s:%hu", host, MQTT_PORT);
	MQTTDisconnect(&c);
	NetworkDisconnect(&n);

	return ret;
}
