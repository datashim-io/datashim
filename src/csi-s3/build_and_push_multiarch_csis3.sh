#!/bin/bash

REGISTRY_URL="${1:-quay.io/datashim-io}"
VERSION="${2:-latest}"
docker buildx build --platform linux/amd64,linux/arm64,linux/ppc64le --network host  --push -t ${REGISTRY_URL}/csi-s3:${VERSION} -f ./cmd/s3driver/Dockerfile .
