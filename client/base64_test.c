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
	void *p;
	char *s;
	size_t out = 0;

	s = yodi_base64_encode("", 0);
	assert(s);
	assert(s[0] == '\0');
	free(s);

	p = yodi_base64_encode("\x01\x02\x03\x04", 4);
	assert(p);
	assert(strcmp(p, "AQIDBA==") == 0);
	free(p);

	assert(!yodi_base64_decode("AQIDBA\x01", 7, &out));
	assert(!yodi_base64_decode("", 0, &out));

	s = yodi_base64_decode("AQIDBA==", 8, &out);
	assert(s);
	assert(out == 4);
	assert(memcmp(s, "\x01\x02\x03\x04", 4) == 0);
	free(s);

	return EXIT_SUCCESS;
}