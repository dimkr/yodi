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

#include <stdlib.h>
#include <signal.h>
#include <errno.h>
#include <limits.h>
#include <string.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/mman.h>
#include <paths.h>
#include <sys/socket.h>
#include <sys/wait.h>

#include <boydemdb.h>
#include <parson.h>

#include <yodi.h>

#define SHELL_BUFSIZ 1024 * 1024
#define SHELL_TIMEOUT 5

static void handle_echo(const JSON_Object *cmd, JSON_Object *result)
{
	const char *data;

	data = json_object_get_string(cmd, "data");

	if (data)
		json_object_set_string(result, "result", data);
	else
		json_object_set_string(result, "error", "no data specified");
}

static void handle_stop(const JSON_Object *cmd, JSON_Object *result)
{
	pid_t ppid;

	ppid = getppid();
	if (ppid == 1) {
		json_object_set_string(result, "error", "cannot kill init");
		return;
	}

	if (kill(ppid, SIGTERM) < 0)
		json_object_set_string(result, "error", strerror(errno));
}

static void handle_log(const JSON_Object *cmd, JSON_Object *result)
{
	struct stat stbuf;
	yodi_autoclose int fd = -1;
	yodi_autofree char *copy = NULL;
	void *p;

	fd = open(YODI_LOG_PATH, O_RDONLY);
	if ((fd < 0) || (fstat(fd, &stbuf) < 0)) {
		json_object_set_string(result, "error", strerror(errno));
		return;
	}

	if (stbuf.st_size == SIZE_MAX) {
		json_object_set_string(result, "error", "");
		return;
	}

	p = mmap(NULL,
	         (size_t)stbuf.st_size + 1,
	         PROT_READ,
	         MAP_PRIVATE,
	         fd,
	         0);
	if (p == MAP_FAILED) {
		json_object_set_string(result, "error", strerror(errno));
		return;
	}

	copy = malloc(stbuf.st_size);
	if (!copy) {
		munmap(p, (size_t)stbuf.st_size);
		json_object_set_string(result, "error", "");
		return;
	}

	memcpy(copy, p, stbuf.st_size);
	munmap(p, (size_t)stbuf.st_size);
	copy[stbuf.st_size] = '\0';

	json_object_set_string(result, "result", copy);
}

static void handle_shell(const JSON_Object *cmd, JSON_Object *result)
{
	struct timeval tv = {.tv_sec = SHELL_TIMEOUT};
	int s[2];
	yodi_autofree char *buf = NULL;
	const char *cmdline;
	size_t len = 0;
	ssize_t chunk;
	pid_t pid;
	yodi_autoclose int our = -1, their = -1;

	cmdline = json_object_get_string(cmd, "cmd");
	if (!cmdline) {
		json_object_set_string(result, "error", "no command specified");
		return;
	}

	buf = malloc(SHELL_BUFSIZ);
	if (!buf)
		return;

	if (socketpair(AF_UNIX, SOCK_STREAM, 0, s) < 0) {
		json_object_set_string(result, "error", strerror(errno));
		return;
	}
	our = s[0];
	their = s[1];

	if (setsockopt(our,
	               SOL_SOCKET,
	               SO_RCVTIMEO,
	               &tv,
	               sizeof(tv)) < 0) {
		json_object_set_string(result, "error", strerror(errno));
		return;
	}

	pid = fork();
	switch (pid) {
	case -1:
		return;

	case 0:
		if ((close(our) < 0) ||
		    (fcntl(their, F_SETFD, FD_CLOEXEC) < 0) ||
		    (dup2(their, STDOUT_FILENO) < 0) ||
		    (close(their) < 0) ||
		    (dup2(STDOUT_FILENO, STDERR_FILENO) < 0) ||
		    (yodi_setsig(STDOUT_FILENO, SIGKILL) < 0) ||
		    (signal(SIGALRM, SIG_DFL) == SIG_ERR))
			exit(EXIT_FAILURE);

		alarm(SHELL_TIMEOUT);

		execl(_PATH_BSHELL, _PATH_BSHELL, "-c", cmdline, (char *)NULL);
		exit(EXIT_FAILURE);
	}
	close(their);
	their = -1;

	while (1) {
		chunk = recv(our, buf + len, SHELL_BUFSIZ - len, 0);
		if (chunk < 0) {
			json_object_set_string(result, "error", strerror(errno));
			break;
		}

		if (chunk == 0) {
			buf[len] = '\0';
			json_object_set_string(result, "result", buf);
			break;
		}

		len += (size_t)chunk;
	}

	waitpid(pid, NULL, WNOHANG);
}

static const struct {
	const char *type;
	void (*handle)(const JSON_Object *, JSON_Object *);
} cmds[] = {
#define CMD(x) {#x, handle_##x}
	CMD(echo),
	CMD(stop),
	CMD(log),
	CMD(shell)
};

static void *run_command(const void *p, const size_t size)
{
	yodi_json_value_autofree JSON_Value *schema = NULL,
	                                    *cmd = NULL,
	                                    *res = NULL;
	JSON_Object *root, *cmdo;
	const char *type, *id;
	char *s = NULL;
	yodi_autofree char *result = NULL;
	unsigned int i;

	schema = json_parse_string("{\"type\":\"\",\"id\":\"\"}");
	if (!schema)
		return s;

	cmd = json_parse_string((const char *)p);
	if (!cmd) {
		return s;
	}

	if (json_validate(schema, cmd) != JSONSuccess) {
		return s;
	}

	res = json_value_init_object();
	if (!res)
		return s;

	cmdo = json_object(cmd);
	type = json_object_get_string(cmdo, "type");
	id = json_object_get_string(cmdo, "id");

	root = json_value_get_object(res);
	if ((json_object_set_string(root, "type", type) != JSONSuccess) ||
	    (json_object_set_string(root, "id", id) != JSONSuccess))
		return s;

	yodi_debug("Running command %.*s",
	           (int)(size % INT_MAX),
	           (char *)p);

	for (i = 0; i < sizeof(cmds) / sizeof(cmds[0]); ++i) {
		if (strcmp(cmds[i].type, type) != 0)
			continue;

		cmds[i].handle(cmdo, root);

		return json_serialize_to_string(res);
	}

	yodi_debug("Unknown command type %s", type);
	return NULL;
}

static void save_result(const void *p, const size_t len, boydemdb db)
{
	yodi_autofree void *comp;
	size_t out;

	comp = yodi_compress(p, len, &out);
	if (comp)
		boydemdb_add(db, YODI_TYPE_RESULT, comp, out);
}


int yodi_worker(int argc, char *argv[])
{
	struct timespec one = {.tv_sec = 1}, zero = {0};
	siginfo_t si;
	sigset_t sigs;
	yodi_db_autoclose boydemdb db = BOYDEMDB_INIT;
	int ret = EXIT_FAILURE;
	boydemdb_id id;
	size_t size;
	void *cmd, *res;

	if ((sigemptyset(&sigs) < 0) || (sigaddset(&sigs, SIGTERM) < 0))
		return EXIT_FAILURE;

	db = boydemdb_open(YODI_DB_PATH);
	if (!db) {
		yodi_error("%s", "Failed to open "YODI_DB_PATH);
		return EXIT_FAILURE;
	}

	while (1) {
		cmd = boydemdb_one(db, YODI_TYPE_COMMAND, &id, &size);
		if (cmd) {
			res = run_command(cmd, size);

			boydemdb_delete(db, id);
			free(cmd);

			if (!res)
				continue;

			yodi_debug("Saving result %s", (char *)res);
			save_result(res, strlen(res), db);
			free(res);
		}

		if (sigtimedwait(&sigs, &si, cmd ? &zero : &one) < 0) {
			if (errno != EAGAIN)
				break;
		}
		else {
			yodi_debug("Received termination signal %d", si.si_signo);
			ret = EXIT_SUCCESS;
			break;
		}
	}

	return ret;
}
