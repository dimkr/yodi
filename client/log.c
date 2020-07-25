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