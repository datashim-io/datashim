#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

function kube_wait(){
	kubectl wait --for=condition=ready pods --all > /dev/null 2>&1
}

function install_noobaa() {
	echo -n "Downloading NooBaa CLI..."
    uKernel="$(uname -s)"
    case "${uKernel}" in
        Darwin*) os=mac;;
        Linux*) os=linux;;
    esac
	wget -P ${DIR} https://github.com/noobaa/noobaa-operator/releases/download/v2.0.10/noobaa-${os}-v2.0.10 > /dev/null 2>&1
	mv ${DIR}/noobaa-${os}-* ${DIR}/noobaa
	chmod +x ${DIR}/noobaa
	echo "done"
	
	echo -n "Installing NooBaa..."
	${DIR}/noobaa install > /dev/null 2>&1
	kube_wait
	${DIR}/noobaa backingstore create pv-pool my-pv-bs --num-volumes 3 --pv-size-gb 1 --storage-class standard > /dev/null 2>&1
	kube_wait
	${DIR}/noobaa bucketclass delete noobaa-default-bucket-class > /dev/null 2>&1
	kube_wait
	${DIR}/noobaa bucketclass create  noobaa-default-bucket-class --backingstores=my-pv-bs --placement="" > /dev/null 2>&1
	echo "done"
}

function build_data_loader() {
	echo -n "Building NooBaa data loader..."
	driver_check=$(cat $HOME/.minikube/machines/minikube/config.json  | grep DriverName)
    if [[ $driver_check != *"none"* ]]; then
      eval $(minikube docker-env)
    fi
	docker build -f ${DIR}/Dockerfile-awscli-alpine -t awscli-alpine . > /dev/null 2>&1
	if [[ $driver_check != *"none"* ]]; then
      eval $(minikube docker-env -u)
    fi
	echo "done"
}

function run_data_loader() {
	echo -n "Creating test OBC..."
	kubectl create -f ${DIR}/obc.yaml >/dev/null 2>&1
	while [ -z "`kubectl get obc | grep Bound`" ]; do sleep 10; done
	echo "done"

	key_id=$(${DIR}/noobaa status 2>&1 | grep AWS_ACCESS_KEY_ID | awk -F ": " '{print $2}')
	acc_key=$(${DIR}/noobaa status 2>&1 | grep AWS_SECRET_ACCESS_KEY | awk -F ": " '{print $2}')
	bucket=$(${DIR}/noobaa bucket list 2>&1 | grep my-bucket)
	
	echo -n "Loading data to example bucket..."
	sed -e "s|{KEY_ID}|${key_id}|g" \
		-e "s|{ACCESS_KEY}|${acc_key}|g" \
		-e "s|{BUCKET}|${bucket}|g" ${DIR}/data-loader-noobaa.yaml | kubectl create -f - > /dev/null 2>&1
	kubectl wait --for=condition=complete  job/example-noobaa-data
	echo "done"
}

install_noobaa
build_data_loader
kube_wait
run_data_loader

