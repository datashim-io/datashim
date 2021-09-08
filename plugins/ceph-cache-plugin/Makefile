SHELL=/bin/bash

define get_arch
$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; elif [ "$$(arch)" == "i386" ]; then echo amd64; else echo "$$(arch)"; fi)
endef

K8S_FILES += $(shell find ./deploy -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./deploy/crds -maxdepth 1 -name '*crd.yaml')

PULL_IMAGE ?= false
IMAGE_NAME ?= quay.io/datashim/ceph-cache-plugin
NAMESPACE ?= dlf
COMMON_IMAGE_TAG ?= latest
IMAGE_NAME := $(IMAGE_NAME):$(COMMON_IMAGE_TAG)-$(call get_arch)

MAKE_ENV += IMAGE_NAME
MAKE_ENV += NAMESPACE

SHELL_EXPORT := $(foreach v,$(MAKE_ENV),$(v)='$($(v))' )


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
		$(SHELL_EXPORT) envsubst < $$file ;\
	done

undeployment:
	@for file in $(K8S_FILES); do \
		echo delete $$file ;\
		kubectl delete -n $(NAMESPACE) -f $$file ;\
	done
