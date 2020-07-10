#include <sys/types.h>
#include <unistd.h>
#include <fcntl.h>

int yodi_setsig(const int fd, const int sig)
{
	int fl;

	fl = fcntl(fd, F_GETFL);
	if ((fl < 0) ||
	    (fcntl(fd, F_SETFL, fl | O_ASYNC) < 0) ||
	    (fcntl(fd, F_SETSIG, sig) < 0) ||
	    (fcntl(fd, F_SETOWN, getpid()) < 0))
		return -1;

	return 0;
}

