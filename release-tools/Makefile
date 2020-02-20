SHELL=/bin/bash

DATASET_OPERATOR_IMAGE := dataset-operator
DATASET_OPERATOR_TAG := latest
DATASET_OPERATOR_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(DATASET_OPERATOR_IMAGE)
DATASET_OPERATOR_IMAGE := $(DATASET_OPERATOR_IMAGE):$(DATASET_OPERATOR_TAG)

CSI_S3_IMAGE := csi-s3
CSI_S3_IMAGE_TAG := dev-full
CSI_S3_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(CSI_S3_IMAGE)
CSI_S3_IMAGE := $(CSI_S3_IMAGE):$(CSI_S3_IMAGE_TAG)

CSI_NFS_IMAGE := csi-nfs
CSI_NFS_IMAGE_TAG := dev-full
CSI_NFS_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(CSI_NFS_IMAGE)
CSI_NFS_IMAGE := $(CSI_NFS_IMAGE):$(CSI_NFS_IMAGE_TAG)

BASE_EXTERNAL_PROVISIONER_IMAGE := csi-provisioner
EXTERNAL_PROVISIONER_IMAGE := $(DOCKER_REGISTRY_SIDECARS)/$(BASE_EXTERNAL_PROVISIONER_IMAGE)
EXTERNAL_PROVISIONER_IMAGE := $(EXTERNAL_PROVISIONER_IMAGE):$(EXTERNAL_PROVISIONER_TAG)

BASE_EXTERNAL_ATTACHER_IMAGE := csi-attacher
EXTERNAL_ATTACHER_IMAGE := $(DOCKER_REGISTRY_SIDECARS)/$(BASE_EXTERNAL_ATTACHER_IMAGE)
EXTERNAL_ATTACHER_IMAGE := $(EXTERNAL_ATTACHER_IMAGE):$(EXTERNAL_ATTACHER_TAG)

BASE_NODE_DRIVER_REGISTRAR_IMAGE := csi-node-driver-registrar
NODE_DRIVER_REGISTRAR_IMAGE := $(DOCKER_REGISTRY_SIDECARS)/$(BASE_NODE_DRIVER_REGISTRAR_IMAGE)
NODE_DRIVER_REGISTRAR_IMAGE := $(NODE_DRIVER_REGISTRAR_IMAGE):$(NODE_DRIVER_REGISTRAR_TAG)

KEY_DIR := $(shell mktemp -d)

MAKE_ENV += EXTERNAL_PROVISIONER_IMAGE
MAKE_ENV += EXTERNAL_ATTACHER_IMAGE
MAKE_ENV += NODE_DRIVER_REGISTRAR_IMAGE
MAKE_ENV += CSI_S3_IMAGE
MAKE_ENV += CSI_NFS_IMAGE
MAKE_ENV += DATASET_OPERATOR_IMAGE
MAKE_ENV += DATASET_OPERATOR_NAMESPACE
MAKE_ENV += DOCKER_REGISTRY_SECRET
MAKE_ENV += KEY_DIR

SHELL_EXPORT := $(foreach v,$(MAKE_ENV),$(v)='$($(v))' )

K8S_FILES += $(shell find ./src/csi-s3/deploy/kubernetes -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/csi-driver-nfs/deploy/kubernetes -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/dataset-operator/deploy -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/dataset-operator/deploy/crds -maxdepth 1 -name '*crd.yaml')

base:
	mkdir -p _tmp
	docker build -t docker-go-base --build-arg ARCH=$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; elif [ "$$(arch)" == "i386" ]; then echo amd64; else echo "$$(arch)"; fi) --build-arg DOCKER_ARCH=$(shell if [ "$$(arch)" == "ppc64le" ]; then echo ppc64el;elif [ "$$(arch)" == "x86_64" ]; then echo amd64; elif [ "$$(arch)" == "i386" ]; then echo amd64; fi) -f ./release-tools/docker-go-base/Dockerfile .

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
	docker run -it -v /var/run/docker.sock:/var/run/docker.sock $(3)-installer make container -e IMAGE_TAG=$(4) GOFLAGS=-mod=vendor
	docker tag $(5):latest $(4)
	rm _tmp/Dockerfile.$(3)
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
	$(call install_sidecar,https://github.com/kubernetes-csi/external-attacher.git,$(EXTERNAL_ATTACHER_TAG),external-attacher,$(EXTERNAL_ATTACHER_IMAGE),$(BASE_EXTERNAL_ATTACHER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/external-provisioner.git,$(EXTERNAL_PROVISIONER_TAG),external-provisioner,$(EXTERNAL_PROVISIONER_IMAGE),$(BASE_EXTERNAL_PROVISIONER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/node-driver-registrar.git,$(NODE_DRIVER_REGISTRAR_TAG),node-driver-registrar,$(NODE_DRIVER_REGISTRAR_IMAGE),$(BASE_NODE_DRIVER_REGISTRAR_IMAGE))
endif
ifeq ($(PULL_COMPONENTS), true)
	@docker pull $(CSI_S3_IMAGE) ;\
	docker pull $(CSI_NFS_IMAGE) ;\
	docker pull $(DATASET_OPERATOR_IMAGE)
else
	$(call install_local,csi-s3,$(CSI_S3_IMAGE))
	$(call install_local,csi-driver-nfs,$(CSI_NFS_IMAGE))
	$(call install_local,dataset-operator,$(DATASET_OPERATOR_IMAGE))
endif

push-containers:
ifeq ($(PULL_SIDECARS), false)
	@docker push $(EXTERNAL_PROVISIONER_IMAGE) ;\
	docker push $(EXTERNAL_ATTACHER_IMAGE) ;\
	docker push $(NODE_DRIVER_REGISTRAR_IMAGE)
endif
ifeq ($(PULL_COMPONENTS), false)
	@docker push $(CSI_S3_IMAGE) ;\
	docker push $(CSI_NFS_IMAGE) ;\
	docker push $(DATASET_OPERATOR_IMAGE)
endif

minikube-load-containers: build-containers
	$(call load_containers_minikube,external-attacher,$(EXTERNAL_ATTACHER_IMAGE))
	$(call load_containers_minikube,external-provisioner,$(EXTERNAL_PROVISIONER_IMAGE))
	$(call load_containers_minikube,node-driver-registrar,$(NODE_DRIVER_REGISTRAR_IMAGE))
	$(call load_containers_minikube,csi-s3,$(CSI_S3_IMAGE))
	$(call load_containers_minikube,csi-driver-nfs,$(CSI_NFS_IMAGE))
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
	@ ./release-tools/generate-keys.sh $(KEY_DIR) $(DATASET_OPERATOR_NAMESPACE);\
	export CA_PEM_B64="$$(openssl base64 -A < "$(KEY_DIR)/ca.crt")" ;\
	$(SHELL_EXPORT) kubectl -n $(DATASET_OPERATOR_NAMESPACE) create secret tls webhook-server-tls \
            --cert "$(KEY_DIR)/webhook-server-tls.crt" \
            --key "$(KEY_DIR)/webhook-server-tls.key" --dry-run -o yaml | kubectl apply -f - ;\
    $(SHELL_EXPORT) envsubst < "./src/dataset-operator/deploy/webhook.yaml.template" | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f -;\
	rm -rf $(KEY_DIR)
