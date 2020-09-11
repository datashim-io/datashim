#!/bin/bash

export ARCH=`arch`;
if [ "$ARCH" == "x86_64" ]; then export ARCH="amd64"; fi
make ARCH=$ARCH DOCKER_REGISTRY_URL=$DOCKER_REGISTRY_URL DOCKER_REGISTRY=$DOCKER_REGISTRY USERNAME=$USERNAME PASSWORD=$PASSWORD build-csi-plugins push-csi-plugins