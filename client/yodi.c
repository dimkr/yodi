#include <pthread.h>
#include <stdlib.h>
#include <stdint.h>
#include <signal.h>
#include <string.h>

#include <boydemdb.h>

#include <yodi.h>

struct client_args {
	char **argv;
	boydemdb db;
	int argc;
};

static void *client_routine(void *p)
{
	const struct client_args *args = (const struct client_args *)p;

	return (void *)(intptr_t)yodi_client(args->argc,
	                                     args->argv);
}

int yodi_main(int argc, char *argv[])
{
	struct client_args args = {.argc = argc, .argv = argv};
	pthread_t client;
	sigset_t set;
	int sig;

	if ((sigemptyset(&set) < 0) ||
	    (sigaddset(&set, SIGINT) < 0) ||
	    (sigaddset(&set, SIGTERM) < 0))
		return EXIT_FAILURE;

	args.db = boydemdb_open("/tmp/x");
	if (!args.db)
		return EXIT_FAILURE;

	if (pthread_create(&client, NULL, client_routine, &args) != 0) {
		boydemdb_close(args.db);
		return EXIT_FAILURE;
	}

	sigwait(&set, &sig);

	pthread_cancel(client);
	pthread_join(client, NULL);

	boydemdb_close(args.db);

	return EXIT_SUCCESS;
}

#include <sys/prctl.h>

const char *yodi_now(void)
{
	static char buf[26];
	time_t now;
	char *newline;

	time(&now);
	ctime_r(&now, buf);

	newline = rindex(buf, '\n');
	if (newline)
		*newline = '\0';

	return buf;
}

#include <sys/prctl.h>

const char *yodi_progname(void)
{
	static char comm[16];
	static pid_t pid;

	if (pid == 0)
		pid = getpid();
	else if ((pid != -1) && (pid != getpid())) {
		pid = -1;
		comm[0] = '\0';
	}

	if (!comm[0])
		prctl(PR_GET_NAME, comm);

	return comm;
}