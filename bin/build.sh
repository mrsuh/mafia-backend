#!/bin/sh

cd "$( cd `dirname $0` && pwd )/.."

if [ ! -f app/config.yml ]; then
   cp app/config.yml.dist app/config.yml
fi

go get github.com/gorilla/websocket
go get github.com/sirupsen/logrus
go get github.com/orcaman/concurrent-map
go get github.com/gorilla/mux
go get github.com/spf13/viper

go build -o bin/server -i mafia-backend/src
