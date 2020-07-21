#ifndef _YODI_H_INCLUDED
#	define _YODI_H_INCLUDED

#	include <stdio.h>
#	include <sys/types.h>
#	include <unistd.h>
#	include <stdlib.h>

#	include <boydemdb.h>
#	include <parson.h>

int yodi_setsig(const int fd, const int sig);

static inline void yodi_autofree_cb(void *p)
{
	if (*(void **)p)
		free(*(void **)p);
}

#	define yodi_autofree __attribute__((cleanup(yodi_autofree_cb)))

static inline void yodi_autoclose_db_cb(void *p)
{
	boydemdb_close(*(boydemdb *)p);
}

#	define yodi_db_autoclose __attribute__((cleanup(yodi_autoclose_db_cb)))

static inline void yodi_json_value_autofree_cb(void *p)
{
	if (*(JSON_Value **)p)
		json_value_free(*(JSON_Value **)p);
}

#	define yodi_json_value_autofree __attribute__((cleanup(yodi_json_value_autofree_cb)))

static inline void yodi_autoclose_cb(void *p)
{
	if (*(int *)p != -1)
		close(*(int *)p);
}

#	define yodi_autoclose __attribute__((cleanup(yodi_autoclose_cb)))

const char *yodi_now(void);
const char *yodi_progname(void);

#	define yodi_warn(fmt, ...) fprintf(stderr, "[ %s | %s/%ld ] "fmt"\n", yodi_now(), yodi_progname(), (long)getpid(), __VA_ARGS__)
#	define yodi_error yodi_warn

#	ifdef YODI_DEBUG
#		define yodi_debug(fmt, ...) do {} while (0)
#	else
#		define yodi_debug yodi_warn
#	endif

#	define YODI_DB_PATH "/tmp/boydem"
#	define YODI_LOG_PATH "/tmp/yodi.log"

enum {
	YODI_TYPE_COMMAND,
	YODI_TYPE_RESULT,
};

int yodi_client(int argc, char *argv[]);
int yodi_worker(int argc, char *argv[]);

#endif
