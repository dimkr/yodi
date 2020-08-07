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

.PHONY: deploy clean

all: build

broker: go.mod go.sum cmd/broker/*.go pkg/*/*.go
	CGO_ENABLED=0 go test -timeout 10s ./...
	CGO_ENABLED=0 go build -ldflags "-s -w" ./cmd/broker

build-broker: deploy/docker/Dockerfile.broker broker
	docker build -f deploy/docker/Dockerfile.broker -t yodi/broker .

mailman: go.mod go.sum cmd/mailman/*.go pkg/*/*.go
	CGO_ENABLED=0 go test -timeout 10s ./...
	CGO_ENABLED=0 go build -ldflags "-s -w" ./cmd/mailman

build-mailman: deploy/docker/Dockerfile.mailman mailman
	docker build -f deploy/docker/Dockerfile.mailman -t yodi/mailman .

client-linux-arm-ssl:
	./client/cross_compile.sh arm-any32-linux-musleabi $@

client-linux-arm:
	./client/cross_compile.sh arm-any32-linux-musleabi $@ -Dssl=false

client-linux-armeb-ssl:
	./client/cross_compile.sh armeb-any32-linux-musleabi $@

client-linux-armeb:
	./client/cross_compile.sh armeb-any32-linux-musleabi $@ -Dssl=false

client-linux-mips-ssl:
	./client/cross_compile.sh mips-any32-linux-musl $@

client-linux-mips:
	./client/cross_compile.sh mips-any32-linux-musl $@ -Dssl=false

client-linux-mipsel-ssl:
	./client/cross_compile.sh mipsel-any32-linux-musl $@

client-linux-mipsel:
	./client/cross_compile.sh mipsel-any32-linux-musl $@ -Dssl=false

client-linux-i386-ssl:
	./client/cross_compile.sh i386-any32-linux-musl $@

client-linux-i386:
	./client/cross_compile.sh i386-any32-linux-musl $@ -Dssl=false

build-client: client-linux-arm-ssl client-linux-arm client-linux-armeb-ssl client-linux-armeb client-linux-mips-ssl client-linux-mips client-linux-mipsel-ssl client-linux-mipsel client-linux-i386-ssl client-linux-i386

web: go.mod go.sum cmd/web/*.go
	CGO_ENABLED=0 go test -timeout 10s ./...
	CGO_ENABLED=0 go build -ldflags "-s -w" ./cmd/web

build-web: deploy/docker/Dockerfile.web build-client web
	docker build -f deploy/docker/Dockerfile.web -t yodi/web .

build: build-broker build-mailman build-web

clean:
	rm -f client-* broker mailman web

deploy: deploy/k8s/*
	kubectl apply -f deploy/k8s -R
	for x in `kubectl get pods -o json | jq -r ".items[].metadata.name"`; do kubectl wait --for=condition=ready --timeout=10s pod/$$x || exit 1; done
	sleep 25 # TODO: why isn't waiting for the pods enough?

start:
	minikube start --disk-size=2gb
	eval $(minikube -p minikube docker-env) && $(MAKE) build
	eval $(minikube -p minikube docker-env) && $(make) deploy

stop:
	minikube delete