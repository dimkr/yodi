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

all: build

build-broker: docker/Dockerfile.broker
	docker build -f docker/Dockerfile.broker -t yodi/broker .

build-mailman: docker/Dockerfile.mailman
	docker build -f docker/Dockerfile.mailman -t yodi/mailman .

client-linux-arm-ssl:
	./build/cross_compile.sh arm-any32-linux-musleabi $@

client-linux-arm:
	./build/cross_compile.sh arm-any32-linux-musleabi $@ -Dssl=false

client-linux-armeb-ssl:
	./build/cross_compile.sh armeb-any32-linux-musleabi $@

client-linux-armeb:
	./build/cross_compile.sh armeb-any32-linux-musleabi $@ -Dssl=false

client-linux-mips-ssl:
	./build/cross_compile.sh mips-any32-linux-musl $@

client-linux-mips:
	./build/cross_compile.sh mips-any32-linux-musl $@ -Dssl=false

client-linux-mipsel-ssl:
	./build/cross_compile.sh mipsel-any32-linux-musl $@

client-linux-mipsel:
	./build/cross_compile.sh mipsel-any32-linux-musl $@ -Dssl=false

client-linux-i386-ssl:
	./build/cross_compile.sh i386-any32-linux-musl $@

client-linux-i386:
	./build/cross_compile.sh i386-any32-linux-musl $@ -Dssl=false

build-client: client-linux-arm-ssl client-linux-arm client-linux-armeb-ssl client-linux-armeb client-linux-mips-ssl client-linux-mips client-linux-mipsel-ssl client-linux-mipsel client-linux-i386-ssl client-linux-i386

build-web: docker/Dockerfile.web build-client
	docker build -f docker/Dockerfile.web -t yodi/web .

build: build-broker build-mailman build-web

clean:
	rm -f client-*

deploy:
	kubectl apply -f k8s -R
	for x in `kubectl get pods -o json | jq -r ".items[].metadata.name"`; do kubectl wait --for=condition=ready --timeout=10s pod/$$x || exit 1; done