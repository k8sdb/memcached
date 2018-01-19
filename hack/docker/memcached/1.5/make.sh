#!/bin/bash
set -xeou pipefail

DOCKER_REGISTRY=${DOCKER_REGISTRY:-kubedb}
IMG=memcached
TAG=1.5
PATCH=1.5.4

docker pull "$DOCKER_REGISTRY/$IMG:$PATCH"

docker tag "$DOCKER_REGISTRY/$IMG:$PATCH" "$DOCKER_REGISTRY/$IMG:$TAG"
docker push "$DOCKER_REGISTRY/$IMG:$TAG"

docker tag "$DOCKER_REGISTRY/$IMG:$TAG" "$DOCKER_REGISTRY/$IMG:$TAG-alpine"
docker push "$DOCKER_REGISTRY/$IMG:$TAG-alpine"
