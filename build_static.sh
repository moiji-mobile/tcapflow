#!/bin/sh

set -ex

apk update
apk add musl-dev gcc go libpcap-dev ca-certificates  git

mkdir /go
export GOPATH=/go
mkdir -p /go/src/github.com/moiji-mobile
mkdir -p /mnt/out
cp -a /mnt /go/src/github.com/moiji-mobile/tcapflow
cd /go/src/github.com/moiji-mobile/tcapflow
rm -f tcapflow*
go get -v ./ ./cmd/tcapflow ./cmd/tcapflow-client ./cmd/tcapflow-server
go build --ldflags '-linkmode external -extldflags "-static"' -v ./cmd/tcapflow
go build --ldflags '-linkmode external -extldflags "-static"' -v ./cmd/tcapflow-client
cp ./tcapflow /mnt/out/
cp ./tcapflow-client /mnt/out/
