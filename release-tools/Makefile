SHELL=/bin/bash

DATASET_OPERATOR_IMAGE := dataset-operator
DATASET_OPERATOR_TAG := latest
ifeq ($(DOCKER_REGISTRY_COMPONENTS), true)
DATASET_OPERATOR_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(DATASET_OPERATOR_IMAGE)
endif
DATASET_OPERATOR_IMAGE := $(DATASET_OPERATOR_IMAGE):$(DATASET_OPERATOR_TAG)

CSI_S3_IMAGE := csi-s3
CSI_S3_IMAGE_TAG := dev-full
ifeq ($(DOCKER_REGISTRY_COMPONENTS), true)
CSI_S3_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(CSI_S3_IMAGE)
endif
CSI_S3_IMAGE := $(CSI_S3_IMAGE):$(CSI_S3_IMAGE_TAG)

EXTERNAL_PROVISIONER_IMAGE := csi-provisioner
ifeq ($(PULL_SIDECARS), true)
EXTERNAL_PROVISIONER_IMAGE := $(DOCKER_REGISTRY_SIDECARS)/$(EXTERNAL_PROVISIONER_IMAGE)
endif
EXTERNAL_PROVISIONER_IMAGE := $(EXTERNAL_PROVISIONER_IMAGE):$(EXTERNAL_PROVISIONER_TAG)

EXTERNAL_ATTACHER_IMAGE := csi-attacher
ifeq ($(PULL_SIDECARS), true)
EXTERNAL_ATTACHER_IMAGE := $(DOCKER_REGISTRY_SIDECARS)/$(EXTERNAL_ATTACHER_IMAGE)
endif
EXTERNAL_ATTACHER_IMAGE := $(EXTERNAL_ATTACHER_IMAGE):$(EXTERNAL_ATTACHER_TAG)

NODE_DRIVER_REGISTRAR_IMAGE := csi-node-driver-registrar
ifeq ($(PULL_SIDECARS), true)
NODE_DRIVER_REGISTRAR_IMAGE := $(DOCKER_REGISTRY_SIDECARS)/$(NODE_DRIVER_REGISTRAR_IMAGE)
endif
NODE_DRIVER_REGISTRAR_IMAGE := $(NODE_DRIVER_REGISTRAR_IMAGE):$(NODE_DRIVER_REGISTRAR_TAG)

MAKE_ENV += EXTERNAL_PROVISIONER_IMAGE
MAKE_ENV += EXTERNAL_ATTACHER_IMAGE
MAKE_ENV += NODE_DRIVER_REGISTRAR_IMAGE
MAKE_ENV += CSI_S3_IMAGE
MAKE_ENV += DATASET_OPERATOR_IMAGE
MAKE_ENV += DATASET_OPERATOR_NAMESPACE

SHELL_EXPORT := $(foreach v,$(MAKE_ENV),$(v)='$($(v))' )

K8S_FILES += $(shell find ./src/csi-s3/deploy/kubernetes -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/dataset-operator/deploy -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/dataset-operator/deploy/crds -maxdepth 1 -name '*crd.yaml')

base:
	docker build -t docker-go-base --build-arg ARCH=$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; else echo "$$(arch)"; fi) --build-arg DOCKER_ARCH=$(shell if [ "$$(arch)" == "ppc64le" ]; then echo ppc64el;elif [ "$$(arch)" == "x86_64" ]; then echo amd64; fi) -f ./release-tools/docker-go-base/Dockerfile .

#1: git repo url
#2: git tag
#3: directory created from pull
#4: image to be created
define install_sidecar
	@if [ ! -d _tmp/$(3) ] ;\
    then \
    	  git clone $(1) _tmp/$(3); \
    fi
	cd _tmp/$(3) ;\
	git checkout $(2)
	printf 'FROM docker-go-base\nCOPY ./$(3) /$(3)  \nWORKDIR /$(3)' > _tmp/Dockerfile.$(3)
	docker build -t $(3)-installer -f ./_tmp/Dockerfile.$(3) ./_tmp
	docker run -it -v /var/run/docker.sock:/var/run/docker.sock $(3)-installer make container -e IMAGE_TAG=$(4)
	rm _tmp/Dockerfile.$(3)
	rm -rf _tmp/$(3)
endef

#1: local directory name
#2: image to be created
define install_local
	printf 'FROM docker-go-base\nCOPY ./$(1) /$(1)  \nWORKDIR /$(1)' > ./src/Dockerfile.$(1)
	docker build -t $(1)-installer -f ./src/Dockerfile.$(1) ./src
	docker run -it -v /var/run/docker.sock:/var/run/docker.sock $(1)-installer make container -e IMAGE_TAG=$(2)
	rm ./src/Dockerfile.$(1)
endef

#1: user friendly name
#2: image to load
define load_containers_minikube
    @docker save $(2) | gzip > _tmp/$(1).tar.gz;\
    eval $$(minikube docker-env) ;\
    docker load < _tmp/$(1).tar.gz ;\
    rm -rf _tmp/$(1).tar.gz
endef

build-containers:
ifeq ($(PULL_SIDECARS), true)
	@docker pull $(EXTERNAL_PROVISIONER_IMAGE) ;\
	docker pull $(EXTERNAL_ATTACHER_IMAGE) ;\
	docker pull $(NODE_DRIVER_REGISTRAR_IMAGE)
else
	$(call install_sidecar,https://github.com/kubernetes-csi/external-attacher.git,$(EXTERNAL_ATTACHER_TAG),external-attacher,$(EXTERNAL_ATTACHER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/external-provisioner.git,$(EXTERNAL_PROVISIONER_TAG),external-provisioner,$(EXTERNAL_PROVISIONER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/node-driver-registrar.git,$(NODE_DRIVER_REGISTRAR_TAG),node-driver-registrar,$(NODE_DRIVER_REGISTRAR_IMAGE))
endif
ifeq ($(PULL_COMPONENTS), true)
	@docker pull $(CSI_S3_IMAGE) ;\
	docker pull $(DATASET_OPERATOR_IMAGE)
else
	$(call install_local,csi-s3,$(CSI_S3_IMAGE))
	$(call install_local,dataset-operator,$(DATASET_OPERATOR_IMAGE))
endif

minikube-load-containers: build-containers
	$(call load_containers_minikube,extrenal-attacher,$(EXTERNAL_ATTACHER_IMAGE))
	$(call load_containers_minikube,extrenal-provisioner,$(EXTERNAL_PROVISIONER_IMAGE))
	$(call load_containers_minikube,node-driver-registrar,$(NODE_DRIVER_REGISTRAR_IMAGE))
	$(call load_containers_minikube,csi-s3,$(CSI_S3_IMAGE))
	$(call load_containers_minikube,dataset-operator,$(DATASET_OPERATOR_IMAGE))

minio-install:
	@kubectl create -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-pvc.yaml?raw=true ;\
    kubectl create -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-deployment.yaml?raw=true ;\
    kubectl create -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-service.yaml?raw=true ;\
    kubectl create -f ./examples/example-minio-data.yaml

minio-uninstall:
	@kubectl delete -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-service.yaml?raw=true ;\
	kubectl delete -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-deployment.yaml?raw=true ;\
	kubectl delete -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-pvc.yaml?raw=true ;\
	kubectl delete -f ./examples/example-minio-data.yaml

keys-installation:
	@keydir="$$(mktemp -d)"; ./release-tools/generate-keys.sh $$keydir; echo $$keydir ;\
	export CA_PEM_B64="$$(openssl base64 -A <"$${keydir}/ca.crt")" ;\
	$(SHELL_EXPORT) kubectl -n $(DATASET_OPERATOR_NAMESPACE) create secret tls webhook-server-tls \
            --cert "$${keydir}/webhook-server-tls.crt" \
            --key "$${keydir}/webhook-server-tls.key" --dry-run -o yaml | kubectl apply -f - ;\
    $(SHELL_EXPORT) envsubst < "./src/dataset-operator/deploy/webhook.yaml.template" | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f -;\
	rm -rf "$$keydir"

minikube-uninstall: minio-uninstall
	@for file in $(K8S_FILES); do \
		$(SHELL_EXPORT) envsubst < $$file | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
    done
	@for namespace in $(NAMESPACES_TO_MONITOR); do \
    	kubectl label namespace $$namespace monitor-pods-datasets- ;\
    done
	@kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) secret webhook-server-tls ;\
	$(SHELL_EXPORT) envsubst < ./src/dataset-operator/deploy/webhook.yaml.template | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f -

deployment:
	@for file in $(K8S_FILES); do \
		$(SHELL_EXPORT) envsubst < $$file | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
	done
	@for namespace in $(NAMESPACES_TO_MONITOR); do \
		kubectl label namespace $$namespace monitor-pods-datasets=enabled ;\
	done

minikube-install: base minio-install minikube-load-containers keys-installation deployment