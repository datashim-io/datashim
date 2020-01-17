#!/bin/bash

NOOBA_HOME="/full/path/to/nooba/installation"

export S3_ENDPOINT=$(minikube service s3 --url | head -n1)


envsubst < conf/hive-site.tmpl | tee conf/hive-site.xml
