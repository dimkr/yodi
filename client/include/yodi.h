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

#ifndef _YODI_H_INCLUDED
#	define _YODI_H_INCLUDED

#	include <yodi/auto.h>
#	include <yodi/log.h>
#	include <yodi/signal.h>
#	include <yodi/db.h>
#	include <yodi/compress.h>
#	include <yodi/base64.h>
#	include <yodi/command.h>
#	include <yodi/cpu.h>

int yodi_client(int argc, char *argv[], struct yodi_cpu_limit *cpu);
int yodi_worker(int argc, char *argv[], struct yodi_cpu_limit *cpu);

#endif
