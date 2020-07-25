```
                 _ _
 _   _  ___   __| (_)
| | | |/ _ \ / _` | |
| |_| | (_) | (_| | |
 \__, |\___/ \__,_|_|
 |___/
```

## Overview

Inspired by [loginsrv](https://github.com/tarent/loginsrv), yodi a collection of backend microservices and a client, that can be used as a building block for an agent-based solution.

The idea to build yodi was born when I worked on a security product that runs on a variety of embedded devices, and there was no self-hosted or semi-cloud-based CI service that allowed me to register devices like routers and single-board computers as nodes that can run my test suite.

The first step towards building a self-hosted CI service that can use any device, is something like yodi.

## Current Status

yodi is in its infancy.

## Planned Features

* A variety of basic commands understood by the client
* Compression of command results using [miniz](https://github.com/richgel999/miniz)
* A HTTP microservice that serves static assets like the client executable, and an installation script that can be piped to the shell in a [curl](https://curl.haxx.se/) one-liner
* Saving of command results in a persistent database
* Crash reporting using [krisa](github.com/dimkr/krisa)

## Implementation

yodi's backend is a partial **and non-conformant** [MQTT v3.1.1](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) broker, implemented in [Go](https://golang.org/). Right now, it supports QoS levels 0 and 1 to some degree and ignores large portions of the standard.

The backend is designed to be horizontally scalable; therefore, data like queued messages is saved in [Redis](https://redis.io/).

The yodi client is implemented in C, using [a fork](https://github.com/dimkr/paho.mqtt.embedded-c/integration-ssl) of [Eclipse Paho MQTT C/C++ client for Embedded platforms](https://github.com/eclipse/paho.mqtt.embedded-c), [mbed TLS](https://tls.mbed.org/), [SQLite](https://www.sqlite.org/), [parson](https://github.com/kgabis/parson) and the [Mozilla CA certificate bundle](https://curl.haxx.se/docs/mk-ca-bundle.html).

The glue that holds all these pieces together is [Meson](https://mesonbuild.com/) and cross-compilation is done using a collection [musl](https://musl.libc.org/)-based [toolchains](https://github.com/dimkr/toolchains).

The client uses a multi-process architecture without use of execve(), to reduce its memory consumption. The executable is extracted to anonymous memory by the [papaw](github.com/dimkr/papaw) stub, so every execve() is expensive.

Communication between the client processes, or between a client process and the backend, is done through a [SQLite](https://www.sqlite.org/) database.

A watchdog takes care of restarting the client processes if they crash, and ensures all client processes are terminated when it stops running, for any reason.

## Credits and Legal Information

yodi is free and unencumbered software released under the terms of the [Apache License Version 2.0](https://www.apache.org/licenses/LICENSE-2.0); see COPYING for the license text.

The ASCII art logo at the top was made using [FIGlet](http://www.figlet.org/).