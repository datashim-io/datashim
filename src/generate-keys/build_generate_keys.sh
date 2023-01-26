#!/bin/bash

REGISTRY_URL="${1:-quay.io/datashim-io}"
docker build -t ${1}/generate-keys .