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

here=`pwd`
cd client

rm -rf $dir
dir=/tmp/build-yodi-client-$1-`echo -n "$3" | md5sum | cut -f 1 -d ' '`
trap "rm -rf $dir" EXIT

meson --cross-file=$1 --buildtype=minsize $3 $dir > /dev/null

cd $here

meson test --print-errorlogs -C $dir

. /opt/x-tools/$1/activate

$1-strip -s -R .note -R .comment $dir/subprojects/papaw/papaw $dir/yodi
python3 $dir/subprojects/papaw/papawify $dir/subprojects/papaw/papaw $dir/yodi $2