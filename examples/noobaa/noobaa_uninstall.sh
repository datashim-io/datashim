#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

kubectl delete pod nginx
kubectl delete obc my-bucket-claim
kubectl delete statefulset my-pv-bs-noobaa-noobaa
${DIR}/noobaa uninstall
kubectl delete pvc noobaastorage-my-pv-bs-noobaa-noobaa-0 noobaastorage-my-pv-bs-noobaa-noobaa-1 noobaastorage-my-pv-bs-noobaa-noobaa-2
kubectl delete job example-noobaa-data
rm ${DIR}/noobaa
rm ${DIR}/dataset-noobaa.yaml
