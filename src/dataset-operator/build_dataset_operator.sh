#!/bin/bash

REGISTRY_URL="${1:-quay.io/datashim-io}"
VERSION="${2:-latest}"
docker build -t ${REGISTRY_URL}/dataset-operator:${VERSION} .