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

#include <assert.h>
#include <stdlib.h>

#include "cpu.c"

int main(int argc, char *argv[])
{
	assert(parse_times("240 (dbus-daemon) S 99 240 240 0 -1 1077936128 775 33 0 0 700", 100) == -1);
	assert(parse_times("240 (dbus-daemon) S 99 240 240 0 -1 1077936128 775 33 0 0 700 500", 100) == 12);
	assert(parse_times("240 (dbus-daemon) S 99 240 240 0 -1 1077936128 775 33 0 0 700 500 0 0 20 0 1 0 1334 7139328 888 18446744073709551615 385165029376 385165256300 549582250752 0 0 0 0 4096 16385 1 0 0 17 3 0 0 0 0 0 385165322920 385165329344 385806852096 549582253388 549582253494 549582253494 549582254051 0", 100) == 12);
	assert(parse_times("240 (dbus-daemon) S 99 240 240 0 -1 1077936128 775 33 0 0 7 5 0 0 20 0 1 0 1334 7139328 888 18446744073709551615 385165029376 385165256300 549582250752 0 0 0 0 4096 16385 1 0 0 17 3 0 0 0 0 0 385165322920 385165329344 385806852096 549582253388 549582253494 549582253494 549582254051 0", 100) == 0);

	return EXIT_SUCCESS;
}