#!/bin/bash
set -xeou pipefail

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=memcached
TAG=1.5.4

docker pull $IMG:$TAG-alpine

docker tag $IMG:$TAG-alpine "$DOCKER_REGISTRY/$IMG:$TAG"
docker push "$DOCKER_REGISTRY/$IMG:$TAG"

docker tag $IMG:$TAG-alpine "$DOCKER_REGISTRY/$IMG:$TAG-alpine"
docker push "$DOCKER_REGISTRY/$IMG:$TAG-alpine"
