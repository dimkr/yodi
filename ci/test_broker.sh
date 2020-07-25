#!/bin/sh -xe

cd client
meson --optimization=3 -Dssl=false build-no-ssl
ninja -C build-no-ssl
cd ..
CGO_ENABLED=0 go build -ldflags "-s -w" ./cmd/broker
PORT=1883 ./broker &
sleep 1
./client/build-no-ssl/yodi -h localhost -i 0b8e29de-13a1-43cf-a793-4d898440550e -u user1 -p password1 &
CGO_ENABLED=0 go build -ldflags "-s -w" ./cmd/mailman
./mailman &
(sleep 1 && mosquitto_pub -u user3 -P password3 -t /0b8e29de-13a1-43cf-a793-4d898440550e/commands -f ci/command.json -q 1) &
mosquitto_sub -u user2 -P password2 -t /0b8e29de-13a1-43cf-a793-4d898440550e/results -W 10 -C 1 > /tmp/result.json
cmp /tmp/result.json ci/result.json