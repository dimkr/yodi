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
#include <stdint.h>

#include <mbedtls/base64.h>

#include <yodi/base64.h>

char *yodi_base64_encode(const void *p, const size_t len)
{
	size_t max;
	unsigned char *buf;
	size_t out;

	mbedtls_base64_encode(NULL, 0, &max, (unsigned char *)p, len);

	if (max == SIZE_MAX)
		return NULL;

	buf = malloc(max + 1);
	if (!buf)
		return NULL;

	if (mbedtls_base64_encode(buf,
	                          max,
	                          &out,
	                          (unsigned char *)p,
	                          len) != 0) {
		free(buf);
		return NULL;
	}
	buf[out] = '\0';

	return (char *)buf;
}

void *yodi_base64_decode(const char *p, const size_t len, size_t *out)
{
	size_t max;
	unsigned char *buf;

	if (len == 0)
		return NULL;

	if (mbedtls_base64_decode(NULL,
	                          0,
	                          &max,
	                          (unsigned char *)p,
	                          len) == MBEDTLS_ERR_BASE64_INVALID_CHARACTER)
		return NULL;

	buf = malloc(max);
	if (!buf)
		return NULL;

	if (mbedtls_base64_decode(buf,
	                          max,
	                          out,
	                          (unsigned char *)p,
	                          len) != 0) {
		free(buf);
		return NULL;
	}

	return buf;
}