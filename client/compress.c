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

#include <miniz.h>

#include <yodi/compress.h>

void *yodi_compress(const void *p, const size_t len, size_t *out)
{
	mz_ulong max;
	unsigned char *buf;

	max = mz_compressBound((mz_ulong)len);

	buf = malloc(max);
	if (!buf)
		return NULL;

	if (mz_compress2(buf,
	                 &max,
	                 (unsigned char *)p,
	                 (mz_ulong)len,
	                 MZ_BEST_SPEED) != MZ_OK) {
		free(buf);
		return NULL;
	}

	*out = (size_t)max;
	return buf;
}

void *yodi_decompress(const void *p, const size_t len, size_t *out)
{
	mz_ulong max;
	unsigned char *buf;

	max = mz_deflateBound(NULL, (mz_ulong)len);

	buf = malloc(max);
	if (!buf)
		return NULL;

	if (mz_uncompress(buf,
	                  &max,
	                  (unsigned char *)p,
	                  (mz_ulong)len) != MZ_OK) {
		free(buf);
		return NULL;
	}

	*out = (size_t)max;
	return buf;
}