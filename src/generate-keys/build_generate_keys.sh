#!/bin/bash

IMAGE_TAG="${IMAGE_TAG:-quay.io/datashim-io/generate-keys}"
docker build -t ${IMAGE_TAG} .