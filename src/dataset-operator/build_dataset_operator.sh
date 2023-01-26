#!/bin/bash

IMAGE_TAG="${IMAGE_TAG:-quay.io/datashim-io/dataset-operator}"
docker build -t ${IMAGE_TAG} .