project('yodi', 'c')

ssl = get_option('ssl')

paho_mqtt_embedded_c = subproject('paho.mqtt.embedded-c',
                                  default_options: [
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

yodi_dependencies = [
	libpaho_embed_mqtt3c_dependency,
	boydemdb_dependency,
	parson_dependency,
]

if host_machine.cpu_family() != 'mips'
	krisa = subproject('krisa')

	krisa_dependency = krisa.get_variable('krisa_dependency')

	yodi_dependencies += krisa_dependency
endif

yodi_srcs = [
	'log.c',
	'signal.c',
	'watchdog.c',
	'client.c',
	'worker.c',
]

yodi_c_args = [
	'-D_GNU_SOURCE',
]

if get_option('buildtype').startswith('debug')
	yodi_c_args += ['-DYODI_DEBUG']
endif

if ssl
	yodi_c_args += ['-DYODI_SSL']
endif

yodi = executable('yodi',
                  yodi_srcs,
                  c_args: yodi_c_args,
                  dependencies: yodi_dependencies)
