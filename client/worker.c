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
#include <string.h>
#include <errno.h>

#include <boydemdb.h>

#include <yodi.h>

int yodi_worker(int argc, char *argv[], struct yodi_cpu_limit *cpu)
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
			res = yodi_run_command(cmd, size);

			boydemdb_delete(db, id);
			free(cmd);

			if (!res)
				continue;

			yodi_debug("Saving result %s", (char *)res);
			boydemdb_add(db, YODI_TYPE_RESULT, res, strlen(res));
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

		yodi_cpu_limit_rearm(cpu);
	}

	return ret;
}
