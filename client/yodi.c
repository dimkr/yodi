#include <time.h>
#include <strings.h>
#include <stdlib.h>
#include <sys/prctl.h>

#include <yodi.h>

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