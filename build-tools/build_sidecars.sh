#!/bin/bash

if [ -z "$DOCKER_REGISTRY" ]
then
      echo "please specify \$DOCKER_REGISTRY"
      exit 1
fi

export ARCH=`arch`;
if [ "$ARCH" == "x86_64" ]; then export ARCH="amd64"; fi
if [ "$ARCH" == "aarch64" ]; then export ARCH="arm64"; fi
make ARCH=$ARCH DOCKER_REGISTRY=$DOCKER_REGISTRY build-sidecars push-sidecars
