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

project('yodi', 'c')

ssl = get_option('ssl')

paho_mqtt_embedded_c = subproject('paho.mqtt.embedded-c',
                                  default_options: [
                                  	'websocket=@0@'.format(get_option('websocket')),
                                  	'ssl=@0@'.format(ssl),
                                  	'ssl_verify=@0@'.format(get_option('ssl_verify')),
                                  	'default_library=static',
                                  ])
libpaho_embed_mqtt3c_dependency = paho_mqtt_embedded_c.get_variable('libpaho_embed_mqtt3c_dependency')

boydemdb = subproject('boydemdb',
                      default_options: [
                      	'default_library=static',
                      ])
boydemdb_dependency = boydemdb.get_variable('boydemdb_dependency')

cmake = import('cmake')

parson = cmake.subproject('parson', cmake_options: [])

parson_dependency = parson.dependency('parson')

papaw = subproject('papaw',
                   default_options: [
                   	'compression=lzma',
                   ])

papaw_dependency = papaw.get_variable('papaw_dependency')

miniz = subproject('miniz')

miniz_dependency = miniz.get_variable('miniz_dependency')

yodi_dependencies = [
	libpaho_embed_mqtt3c_dependency,
	boydemdb_dependency,
	parson_dependency,
	papaw_dependency,
	miniz_dependency,
]

yodi_c_args = [
	'-D_GNU_SOURCE',
]

if ssl
	yodi_c_args += ['-DYODI_SSL']
endif

if host_machine.cpu_family() != 'mips'
	krisa = subproject('krisa')

	krisa_dependency = krisa.get_variable('krisa_dependency')

	yodi_dependencies += krisa_dependency

	yodi_c_args += ['-DYODI_HAVE_KRISA']
endif

if get_option('debug')
	yodi_c_args += ['-DYODI_DEBUG']
endif

yodi_srcs = [
	'log.c',
	'signal.c',
	'client.c',
	'worker.c',
	'compress.c',
	'base64.c',
	'command.c',
	'cpu.c',
]

yodi_tests = {
	'compress': [
		'compress.c',
		'compress_test.c',
	],
	'base64': [
		'base64.c',
		'base64_test.c',
	],
	'command': [
		'log.c',
		'signal.c',
		'base64.c',
		'compress.c',
		'command.c',
		'command_test.c',
	],
	'cpu': [
		'cpu_test.c',
	],
}

yodi_includes = include_directories('include')

yodi = executable('yodi',
                  yodi_srcs,
                  'watchdog.c',
                  c_args: yodi_c_args,
                  dependencies: yodi_dependencies,
                  include_directories: yodi_includes)

foreach name, srcs: yodi_tests
	exe = executable('@0@_test'.format(name),
	                 srcs,
	                 c_args: yodi_c_args,
	                 dependencies: yodi_dependencies,
	                 include_directories: yodi_includes)
	test(name, exe)
endforeach
