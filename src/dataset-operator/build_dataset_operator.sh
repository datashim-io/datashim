#!/bin/bash

REGISTRY_URL="${REGISTRY_URL:-quay.io/datashim-io}"
docker build -t ${REGISTRY_URL}/dataset-operator .