#!/bin/bash

MAKE_ARGS="build-components"

if [ -n "$DOCKER_REGISTRY" ]
then
      MAKE_ARGS+=" push-components"
else
      DOCKER_REGISTRY="local"
fi

export ARCH=`arch`;
if [ "$ARCH" == "x86_64" ]; then export ARCH="amd64"; fi
if [ "$ARCH" == "i386" ]; then export ARCH="amd64"; fi
if [ "$ARCH" == "aarch64" ]; then export ARCH="arm64"; fi
make ARCH=$ARCH DOCKER_REGISTRY=$DOCKER_REGISTRY $MAKE_ARGS
