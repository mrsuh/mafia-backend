#!/bin/sh

cd "$( cd `dirname $0` && pwd )/.."

go get github.com/gorilla/websocket
go get github.com/sirupsen/logrus
go get github.com/gorilla/mux
go get github.com/spf13/viper

go build -ldflags "-linkmode external -extldflags -static" -o bin/server -i mafia-backend/src
