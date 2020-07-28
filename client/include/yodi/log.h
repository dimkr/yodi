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
#include <sys/types.h>
#include <unistd.h>

#define YODI_LOG_PATH "/tmp/yodi.log"

const char *yodi_now(void);
const char *yodi_progname(void);

#define yodi_warn(fmt, ...) fprintf(stderr, "[ %s | %s/%ld ] "fmt"\n", yodi_now(), yodi_progname(), (long)getpid(), __VA_ARGS__)
#define yodi_error yodi_warn

#ifdef YODI_DEBUG
#   define yodi_debug yodi_warn
#else
#   define yodi_debug(fmt, ...) do {} while (0)
#endif