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

#include <stdio.h>
#include <limits.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <signal.h>

#include <yodi/cpu.h>

/*
 * when called at intervals of >= REARM_INTERVAL seconds, yodi_cpu_limit_rearm()
 * sets the process CPU time limit to number of CPU seconds consumed so far,
 * plus CPU_SEC
 */
#define CPU_SEC 110
#define REARM_INTERVAL 120

static int parse_times(const char *stat, const long ticks)
{
	unsigned long utime, stime;

	if (sscanf(stat,
	           "%*d %*s %*s %*d %*d %*d %*d %*d %*d %*d %*d %*d %*d %lu %lu",
	           &utime,
	           &stime) != 2)
		return -1;

	if (utime > INT_MAX - stime)
		return -1;

	return (int)(utime + stime) / ticks;
}

static int cpu_now(void)
{
	static char buf[512];
	ssize_t len;
	long ticks;
	int fd;

	ticks = sysconf(_SC_CLK_TCK);
	if (ticks <= 0)
		return -1;

	fd = open("/proc/self/stat", O_RDONLY);
	if (fd < 0)
		return -1;
	len = read(fd, buf, sizeof(buf) - 1);
	close(fd);
	if (len <= 0)
		return -1;
	buf[len] = '\0';

	return parse_times(buf, ticks);
}

static void yodi_cpu_limit_do_rearm(struct yodi_cpu_limit *lim)
{
	struct rlimit rlim = {.rlim_max = RLIM_INFINITY};
	unsigned int now;

	now = cpu_now();
	if ((now < 0) || (now >= UINT_MAX - CPU_SEC))
		return;

	rlim.rlim_cur = (rlim_t)(now + CPU_SEC);

	setrlimit(RLIMIT_CPU, &rlim);
}

void yodi_cpu_limit_arm(struct yodi_cpu_limit *lim)
{
	TimerInit(&lim->timer);
	TimerCountdown(&lim->timer, REARM_INTERVAL);

	signal(SIGXCPU, SIG_DFL);

	yodi_cpu_limit_do_rearm(lim);
}

void yodi_cpu_limit_rearm(struct yodi_cpu_limit *lim)
{
	if (TimerIsExpired(&lim->timer)) {
		yodi_cpu_limit_do_rearm(lim);
		TimerCountdown(&lim->timer, REARM_INTERVAL);
	}
}
