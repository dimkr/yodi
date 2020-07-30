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

#include <string.h>
#include <assert.h>
#include <stdlib.h>

#include <yodi.h>

int main(int argc, char *argv[])
{
	static const char expr[] = "{\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\", \"type\": \"shell\", \"cmd\": \"expr 1 + 4\"}",
	                  bad_json[] = "{\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\", \"cmd\": \"expr 1 + 4\"",
	                  no_type[] = "{\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\", \"cmd\": \"expr 1 + 4\"}",
	                  no_id[] = "{\"type\": \"shell\", \"cmd\": \"expr 1 + 4\"}",
	                  bad_type[] = "{\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\", \"type\": \"shelll\", \"cmd\": \"expr 1 + 4\"}",
	                  id_int[] = "{\"id\":1, \"type\": \"shell\", \"cmd\": \"expr 1 + 4\"}",
	                  no_cmd[] = "{\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\", \"type\": \"shell\"}",
	                  cmd_int[] = "{\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\", \"type\": \"shell\", \"cmd\": 1}";
	void *p;

	assert(!yodi_run_command("", 0));
	assert(!yodi_run_command(bad_json, sizeof(bad_json) - 1));
	assert(!yodi_run_command(no_type, sizeof(no_type) - 1));
	assert(!yodi_run_command(no_id, sizeof(no_id) - 1));
	assert(!yodi_run_command(bad_type, sizeof(bad_type) - 1));
	assert(!yodi_run_command(id_int, sizeof(id_int) - 1));

	p = yodi_run_command(expr, sizeof(expr) - 1);
	assert(p);
	assert(strcmp((char *)p, "{\"type\":\"shell\",\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\",\"result\":\"eAEBAgD9\\/zUKAHYAQA==\"}") == 0);
	free(p);

	p = yodi_run_command(no_cmd, sizeof(no_cmd) - 1);
	assert(p);
	puts(p);
	assert(strcmp((char *)p, "{\"type\":\"shell\",\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\",\"error\":\"no command specified\"}") == 0);
	free(p);

	p = yodi_run_command(cmd_int, sizeof(cmd_int) - 1);
	assert(p);
	assert(strcmp((char *)p, "{\"type\":\"shell\",\"id\":\"4b1652a9-cf4a-4212-b5e9-09472954de98\",\"error\":\"no command specified\"}") == 0);
	free(p);

	return EXIT_SUCCESS;
}