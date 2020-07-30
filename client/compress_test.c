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
	void *p, *d;
	size_t out = 0;

	p = yodi_compress("", 0, &out);
	assert(p);
	assert(out > 0);
	free(p);

	assert(!yodi_decompress("", 0, &out));

	out = 0;
	p = yodi_compress("\x01\x02\x03\x04", 4, &out);
	assert(p);
	assert(out > 4);

	d = yodi_decompress(p, out, &out);
	assert(d);
	assert(out == 4);
	assert(memcmp(d, "\x01\x02\x03\x04", 4) == 0);
	free(d);

	free(p);

	return EXIT_SUCCESS;
}