#!/bin/bash

make DOCKER_REGISTRY=$DOCKER_REGISTRY USERNAME=$USERNAME PASSWORD=$PASSWORD build-sidecars build-components push-all