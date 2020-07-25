#!/bin/sh -e

# This file is part of yodi.
#
# Copyright 2020 Dima Krasner
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

meson --cross-file=$1 --buildtype=minsize build-$1
ninja -C build-$1

. /opt/x-tools/$1/activate

$1-strip -s -R .note -R .comment build-$1/subprojects/papaw/papaw build-$1/yodi
python3 build-$1/subprojects/papaw/papawify build-$1/subprojects/papaw/papaw build-$1/yodi $2