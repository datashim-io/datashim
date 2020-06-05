SHELL=/bin/bash

K8S_FILES += $(shell find ./deploy -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./deploy/crds -maxdepth 1 -name '*crd.yaml')

PULL_IMAGE ?= false
IMAGE_NAME ?= ceph-cache-plugin
NAMESPACE ?= default

MAKE_ENV += IMAGE_NAME
MAKE_ENV += NAMESPACE

SHELL_EXPORT := $(foreach v,$(MAKE_ENV),$(v)='$($(v))' )

define get_arch
$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; elif [ "$$(arch)" == "i386" ]; then echo amd64; else echo "$$(arch)"; fi)
endef

define load_containers_minikube
    @driver_check=$$(cat $$HOME/.minikube/machines/minikube/config.json  | grep DriverName);\
    if [[ $$driver_check != *"none"* ]]; then docker save ${IMAGE_NAME} | gzip > ./_tmp/ceph-cache-plugin.tar.gz;\
    eval $$(minikube docker-env);\
    docker load < ./_tmp/ceph-cache-plugin.tar.gz;\
    eval $$(minikube docker-env -u);\
    rm -rf ./_tmp/ceph-cache-plugin.tar.gz;\
    fi;
endef

build-container:
	docker build -t ${IMAGE_NAME} --build-arg ARCH=$(call get_arch) .

build-load-container: build-container
	$(call load_containers_minikube)

deployment:
	@for file in $(K8S_FILES); do \
		echo creating $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl apply -n $(NAMESPACE) -f - ;\
	done