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
#include <stdlib.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/wait.h>
#include <errno.h>
#include <string.h>
#include <time.h>
#include <sys/prctl.h>
#include <stdio.h>
#include <sched.h>
#include <sys/time.h>
#include <sys/resource.h>

#ifdef YODI_HAVE_KRISA
#	include <krisa.h>
#endif
#include <papaw.h>

#include <yodi.h>

#define SIGLOG SIGRTMAX
#define SIGRESTART SIGRTMIN
#define KILLFD 127
#define BACKTRACE_SIZE 4096

struct yodi_service {
	int (*fn)(int, char **, struct yodi_cpu_limit *);
	const char *name;
	pid_t pid;
	int killfd;
};

static pid_t start_service(struct yodi_service *svcs,
                           const unsigned int i,
                           const unsigned int n,
                           const int argc,
                           char **argv,
                           const int logr,
                           const int logw,
                           const long delay)
{
	struct yodi_cpu_limit cpu;
	struct timespec ts = {.tv_sec = delay};
	struct yodi_service *svc = &svcs[i];
	int s[2], killfd;
	unsigned int j;

	yodi_debug("Starting %s", svc->name);

	if (socketpair(AF_UNIX, SOCK_STREAM, 0, s) < 0)
		return -1;

	if ((yodi_setsig(s[1], SIGRESTART + i) < 0) ||
	    (fcntl(s[0], F_SETFD, FD_CLOEXEC) < 0)) {
		close(s[1]);
		close(s[0]);
		return -1;
	}

	svc->pid = fork();
	switch (svc->pid) {
	case 0:
		close(s[1]);

		killfd = dup2(s[0], KILLFD);
		close(s[0]);

		close(logr);

		for (j = 0; j < n; ++j) {
			if ((j != i) && (svcs[j].killfd != -1))
				close(svcs[j].killfd);
		}

		if ((killfd != KILLFD) ||
		    (yodi_setsig(killfd, SIGKILL) < 0) ||
		    (fcntl(killfd, F_SETFD, FD_CLOEXEC) < 0) ||
		    (dup2(logw, STDERR_FILENO) != STDERR_FILENO)) {
			if (ts.tv_sec > 0)
				nanosleep(&ts, NULL);
			exit(EXIT_FAILURE);
		}

		close(logw);

		prctl(PR_SET_NAME, svc->name);

		if (ts.tv_sec > 0)
			nanosleep(&ts, NULL);

		yodi_cpu_limit_arm(&cpu);

		exit(svc->fn(argc, argv, &cpu));

	case -1:
		close(s[1]);
	}

	close(s[0]);
	svc->killfd = s[1];
	return svc->pid;
}

static void save_backtrace(boydemdb db, const char *name, const int fd)
{
#ifdef YODI_HAVE_KRISA
	yodi_autofree char *buf = NULL;
	ssize_t len, total = 0;

	buf = malloc(BACKTRACE_SIZE);
	if (!buf)
		return;

	do {
		len = recv(fd, &buf[total], BACKTRACE_SIZE - total, MSG_DONTWAIT);
		if (len < 0) {
			if ((errno == EAGAIN) || (errno == EWOULDBLOCK))
				break;

			return;
		}

		if (len == 0)
			break;

		total += (size_t)len;
	} while (total < BACKTRACE_SIZE);

	if (total > 0)
		boydemdb_add(db, YODI_TYPE_BACKTRACE, buf, total);
#endif
}

static void reap_service(struct yodi_service *svc,
                         const sigset_t *set,
                         boydemdb db)
{
	struct timespec ts = {.tv_sec = 1};
	siginfo_t si;
	int status;

	if (kill(svc->pid, SIGTERM) == 0) {
		while ((sigtimedwait(set, &si, &ts) == 0) &&
		       (si.si_pid != svc->pid))
			waitpid(si.si_pid, NULL, WNOHANG);
	}

	save_backtrace(db, svc->name, svc->killfd);

	close(svc->killfd);
	if (waitpid(svc->pid, &status, 0) == svc->pid) {
		if (WIFEXITED(status))
			yodi_warn("%s has exited with status %d", svc->name, WEXITSTATUS(status));
		else if (WIFSIGNALED(status) && (WTERMSIG(status) == SIGXCPU))
			yodi_warn("%s has exceeded the CPU limit", svc->name);
		else if (WIFSIGNALED(status))
			yodi_warn("%s was terminated by signal %d", svc->name, WTERMSIG(status));
		else
			yodi_warn("%s has terminated for an unknown reason", svc->name);
	}
	svc->pid = -1;
}

static void unqueue_signal(const int sig)
{
	struct timespec ts = {.tv_sec = 0};
	siginfo_t si;
	sigset_t set;

	if ((sigemptyset(&set) < 0) || (sigaddset(&set, sig) < 0))
		return;

	while (sigtimedwait(&set, &si, &ts) > 0);
}

static int wait_for_signal(struct yodi_service *svc,
                           const unsigned int n,
                           const sigset_t *mask)
{
	siginfo_t si;

	while (1) {
		if (sigwaitinfo(mask, &si) < 0)
			return SIGTERM;

		if (si.si_signo == SIGCHLD)
			waitpid(si.si_pid, NULL, WNOHANG);
		else if ((si.si_signo == SIGINT) ||
		         (si.si_signo == SIGTERM) ||
		         (si.si_signo == SIGLOG) ||
		         ((si.si_signo >= SIGRESTART) &&
		          (si.si_signo < SIGRESTART + n)))
			return si.si_signo;
	}

	return si.si_signo;
}

static void save_log(const int fd, boydemdb db)
{
	static char buf[512];
	ssize_t len;

	while (1) {
		len = recvfrom(fd, buf, sizeof(buf), MSG_DONTWAIT, NULL, NULL);
		if (len <= 0)
			break;

		if (len > 1)
			boydemdb_add(db, YODI_TYPE_LOG, buf, (size_t)len - 1);

		write(STDERR_FILENO, buf, (size_t)len);
	}
}

#ifdef YODI_HAVE_KRISA

static int get_killfd(void)
{
	return KILLFD;
}

#endif

#define YODI_SVC(x) {.fn = yodi_##x, .name = #x}

int main(int argc, char *argv[])
{
	static struct yodi_service svcs[] = {
		YODI_SVC(client),
		YODI_SVC(worker),
	};
	struct sched_param sched = {.sched_priority = 0};
	sigset_t mask, chld;
	yodi_db_autoclose boydemdb db = BOYDEMDB_INIT;
	int fds[2];
	pid_t pid;
	unsigned int i, n = sizeof(svcs) / sizeof(svcs[0]), id;
	int sig, ret = EXIT_FAILURE;
	yodi_autoclose int logr = -1, logw = -1;
#ifdef YODI_HAVE_KRISA
	yodi_autoclose int killfd = -1;
#endif

	if ((SIGRESTART + n >= SIGRTMAX) ||
	    (sigemptyset(&mask) < 0) ||
	    (sigaddset(&mask, SIGTERM) < 0) ||
	    (sigaddset(&mask, SIGINT) < 0) ||
	    (sigaddset(&mask, SIGLOG) < 0) ||
	    (sigemptyset(&chld) < 0) ||
	    (sigaddset(&chld, SIGCHLD) < 0))
		return EXIT_FAILURE;

	for (i = 0; i < n; ++i) {
		if (sigaddset(&mask, SIGRESTART + i) < 0)
			return EXIT_FAILURE;
	}

	if (sigprocmask(SIG_SETMASK, &mask, NULL) < 0)
		return EXIT_FAILURE;

	pid = getpid();

	if (sched_setscheduler(pid, SCHED_OTHER, &sched) != 0)
		return EXIT_FAILURE;

	if (setpriority(PRIO_PROCESS, (int)pid, 0) < 0)
		return EXIT_FAILURE;

	papaw_hide_exe();

#ifdef YODI_HAVE_KRISA
	killfd = dup2(STDERR_FILENO, KILLFD);
	krisa_init(get_killfd);
#endif

#ifdef YODI_DEBUG
	if (!isatty(STDERR_FILENO)) {
#else
	if (1) {
#endif
		close(STDERR_FILENO);
		if (open(YODI_LOG_PATH, O_WRONLY | O_CREAT | O_TRUNC, 0600) < 0)
			return EXIT_FAILURE;
		setlinebuf(stderr);
	}

	db = boydemdb_open(YODI_DB_PATH);
	if (!db)
		return EXIT_FAILURE;

	if (socketpair(AF_UNIX, SOCK_DGRAM, 0, fds) < 0)
		return EXIT_FAILURE;
	logr = fds[0];
	logw = fds[1];

	if ((yodi_setsig(logr, SIGLOG) < 0) ||
	    (fcntl(logr, F_SETFD, FD_CLOEXEC) < 0) ||
	    (fcntl(logw, F_SETFD, FD_CLOEXEC) < 0))
		return EXIT_FAILURE;

	for (i = 0; i < n; ++i)
		svcs[i].pid = -1;

	for (i = 0; i < n; ++i) {
		if (start_service(svcs, i, n, argc, argv, logr, logw, 0) < 0)
			goto reap;
	}

	while (1) {
		sig = wait_for_signal(svcs, n, &mask);

		if (sig == SIGLOG) {
			save_log(logr, db);
			continue;
		}

		if ((sig == SIGINT) || (sig == SIGTERM)) {
			yodi_debug("Received termination signal %d", sig);
			ret = EXIT_SUCCESS;
			break;
		}

		id = sig - SIGRESTART;

		reap_service(&svcs[id], &chld, db);
		unqueue_signal(sig);

		if (start_service(svcs, id, n, argc, argv, logr, logw, 1) < 0)
			break;
	}

reap:
	for (i = 0; i < n; ++i) {
		if (svcs[i].pid != -1) {
			yodi_debug("Stopping %s", svcs[i].name);
			reap_service(&svcs[i], &chld, db);
		}
	}

	unlink(YODI_DB_PATH);

	return ret;
}
