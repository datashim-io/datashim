#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

function check_minikube_version() {
  is_correct_version=$(minikube config get kubernetes-version | grep "v1.17")
  if [ -z "$is_correct_version" ]; then
    echo "Minikube uses incompatible k8s version"
    echo "Execute 'minikube config set kubernetes-version v1.17.8' and restart minikube"
    exit 0
  fi
}

function wait_for_backingstore_ready() {
  for (( ; ; )); do
    backingstore_ready=$(kubectl get backingstore -o=jsonpath='{.items[0].status.phase}')
    if [[ $backingstore_ready == "Ready" ]]; then
      echo "Backingstore ready!"
      break
    fi
    echo "Waiting for backingstore to be ready"
    sleep 5
  done
}

function wait_for_pods_created() {
  for (( ; ; )); do
    noobaa_pods=$(kubectl get pods --no-headers -l app=noobaa | wc -l)
    if [[ $noobaa_pods == "2" ]]; then
      echo "Noobaa pods ready!"
      break
    fi
    echo "Waiting for the 2 noobaa pods to be ready"
    sleep 5
  done
}

function wait_for_pods_running() {
  kubectl wait --for=condition=ready pods -l app=noobaa >/dev/null 2>&1
}

function install_noobaa() {
  echo -n "Downloading NooBaa CLI..."
  uKernel="$(uname -s)"
  case "${uKernel}" in
  Darwin*) os=mac ;;
  Linux*) os=linux ;;
  esac
  wget -P ${DIR} https://github.com/noobaa/noobaa-operator/releases/download/v2.0.10/noobaa-${os}-v2.0.10 >/dev/null 2>&1
  mv ${DIR}/noobaa-${os}-* ${DIR}/noobaa
  chmod +x ${DIR}/noobaa
  echo "done"

  echo "Installing NooBaa..."
  ${DIR}/noobaa install >/dev/null 2>&1
  wait_for_pods_created
  wait_for_pods_running
  echo "Installed NooBaa"
  echo "Creating Backing Store"
  ${DIR}/noobaa backingstore create pv-pool my-pv-bs --num-volumes 3 --pv-size-gb 1 --storage-class standard >/dev/null 2>&1
  echo "Created Backing Store"
  wait_for_backingstore_ready
  echo "Delete Bucket Class"
  ${DIR}/noobaa bucketclass delete noobaa-default-bucket-class >/dev/null 2>&1
  echo "Delete Bucket Class"
  echo "Creating Bucket Class"
  ${DIR}/noobaa bucketclass create noobaa-default-bucket-class --backingstores=my-pv-bs --placement="" >/dev/null 2>&1
  echo "Created Bucket Class"
  echo "done"
}

function build_data_loader() {
  echo -n "Building NooBaa data loader..."
  driver_check=$(cat $HOME/.minikube/machines/minikube/config.json | grep DriverName)
  if [[ $driver_check != *"none"* ]]; then
    eval $(minikube docker-env)
  fi
  docker build -f ${DIR}/Dockerfile-awscli-alpine -t awscli-alpine . >/dev/null 2>&1
  if [[ $driver_check != *"none"* ]]; then
    eval $(minikube docker-env -u)
  fi
  echo "done"
}

function run_data_loader() {
  echo -n "Creating test OBC..."
  kubectl create -f ${DIR}/obc.yaml >/dev/null 2>&1
  while [ -z "$(kubectl get obc | grep Bound)" ]; do sleep 10; done
  echo "done"

  key_id=$(${DIR}/noobaa status 2>&1 | grep AWS_ACCESS_KEY_ID | awk -F ": " '{print $2}')
  acc_key=$(${DIR}/noobaa status 2>&1 | grep AWS_SECRET_ACCESS_KEY | awk -F ": " '{print $2}')
  bucket=$(${DIR}/noobaa bucket list 2>&1 | grep my-bucket)

  echo -n "Loading data to example bucket..."
  sed -e "s|{KEY_ID}|${key_id}|g" \
    -e "s|{ACCESS_KEY}|${acc_key}|g" \
    -e "s|{BUCKET}|${bucket}|g" ${DIR}/data-loader-noobaa.yaml | kubectl create -f - >/dev/null 2>&1
  kubectl wait --for=condition=complete job/example-noobaa-data
  echo "done"
}

check_minikube_version
install_noobaa
build_data_loader
run_data_loader
