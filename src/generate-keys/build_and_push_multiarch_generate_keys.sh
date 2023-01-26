#!/bin/bash

IMAGE_TAG="${IMAGE_TAG:-quay.io/datashim-io/generate-keys}"
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64,linux/ppc64le --push -t ${IMAGE_TAG} .