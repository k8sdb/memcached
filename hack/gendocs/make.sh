#!/usr/bin/env bash

pushd $GOPATH/src/github.com/k8sdb/memcached/hack/gendocs
go run main.go
popd
