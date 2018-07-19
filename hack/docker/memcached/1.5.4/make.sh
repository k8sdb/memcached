#!/bin/bash
set -xeou pipefail

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
REPO_ROOT=$GOPATH/src/github.com/kubedb/memcached

IMG=memcached
TAG=1.5.4

# build docker image
pushd $REPO_ROOT/hack/docker/memcached/$TAG
chmod +x start.sh
cmd="docker build -t $DOCKER_REGISTRY/$IMG:$TAG ."
echo $cmd; $cmd

docker push "$DOCKER_REGISTRY/$IMG:$TAG"

docker tag $DOCKER_REGISTRY/$IMG:$TAG "$DOCKER_REGISTRY/$IMG:$TAG-alpine"
docker push "$DOCKER_REGISTRY/$IMG:$TAG-alpine"
