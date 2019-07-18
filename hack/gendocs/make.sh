#!/usr/bin/env bash

pushd $GOPATH/src/kubedb.dev/memcached/hack/gendocs
go run main.go
popd
