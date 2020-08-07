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

build-web: docker/Dockerfile.web
	docker build -f docker/Dockerfile.web -t yodi/web .

build: build-broker build-mailman build-web

deploy:
	kubectl apply -f k8s -R
	for x in `kubectl get pods -o json | jq -r ".items[].metadata.name"`; do kubectl wait --for=condition=ready pod/$$x || exit 1; done