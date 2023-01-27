#!/bin/bash

REGISTRY_URL="${1:-quay.io/datashim-io}"
docker buildx build --platform linux/amd64,linux/arm64,linux/ppc64le --push -t ${REGISTRY_URL}/dataset-operator .