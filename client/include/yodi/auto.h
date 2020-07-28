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

#include <stdlib.h>
#include <boydemdb.h>
#include <parson.h>

static inline void yodi_autofree_cb(void *p)
{
	if (*(void **)p)
		free(*(void **)p);
}

#define yodi_autofree __attribute__((cleanup(yodi_autofree_cb)))

static inline void yodi_autoclose_db_cb(void *p)
{
	boydemdb_close(*(boydemdb *)p);
}

#define yodi_db_autoclose __attribute__((cleanup(yodi_autoclose_db_cb)))

static inline void yodi_json_value_autofree_cb(void *p)
{
	if (*(JSON_Value **)p)
		json_value_free(*(JSON_Value **)p);
}

#define yodi_json_value_autofree __attribute__((cleanup(yodi_json_value_autofree_cb)))

static inline void yodi_autoclose_cb(void *p)
{
	if (*(int *)p != -1)
		close(*(int *)p);
}

#define yodi_autoclose __attribute__((cleanup(yodi_autoclose_cb)))