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

# returns arch as amd64,ppc64le, mostly needed for golang
define get_arch
$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; elif [ "$$(arch)" == "i386" ]; then echo amd64; else echo "$$(arch)"; fi)
endef
#1: git repo url
#2: git tag
#3: directory created from pull
#4: image to be created
define install_sidecar
	@if [ ! -d ./release-tools/_tmp/$(3) ] ;\
    then \
    	  git clone $(1) ./release-tools/_tmp/$(3); \
    fi
	cd ./release-tools/_tmp/$(3) ;\
	git checkout $(2)
	cp ./release-tools/Dockerfile ./release-tools/_tmp/$(3)/Dockerfile-$(3).installer
	echo -e "\nRUN mkdir /tmp-code\
	\nCOPY go.mod /tmp-code/go.mod\
	\nCOPY go.sum /tmp-code/go.sum\
	\nRUN cd /tmp-code && go mod download\
	\nCOPY . /$(3)\
	\nRUN cd /$(3) && make build" | tee -a ./release-tools/_tmp/$(3)/Dockerfile-$(3).installer ;\
	docker build --build-arg=ARCH=$(call get_arch) -t $(3)-installer -f ./release-tools/_tmp/$(3)/Dockerfile-$(3).installer ./release-tools/_tmp/$(3) ;\
	mkdir -p ./release-tools/_tmp/$(3)/bin ;\
	docker run --rm -v $$(pwd)/release-tools/_tmp/$(3)/bin:/tmp-bin $(3)-installer sh -c "cp -r /$(3)/bin/* /tmp-bin" ;\
	cd ./release-tools/_tmp/$(3) &&	docker build -t $(4) .
endef

#1: local directory name
#2: image to be created
define install_local
	cd src/$(1) && make container -e IMAGE_TAG=$(2) -e ARCH=$(call get_arch)
endef

#1: user friendly name
#2: image to load
define load_containers_minikube
    @if [[ $$driver_check == *"virtualbox"* ]]; then docker save $(2) | gzip > ./release-tools/_tmp/$(1).tar.gz;\
    eval $$(minikube docker-env -u);\
    docker load < ./release-tools/_tmp/$(1).tar.gz ;\
    rm -rf ./release-tools/_tmp/$(1).tar.gz;\
    fi;\
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

noobaa-install:
	./examples/noobaa/noobaa_install.sh

noobaa-uninstall:
	./examples/noobaa/noobaa_uninstall.sh

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
    kubectl create -f ./examples/templates/example-minio-data.yaml

minio-uninstall:
	@kubectl delete -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-service.yaml?raw=true ;\
	kubectl delete -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-deployment.yaml?raw=true ;\
	kubectl delete -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes/minio-standalone-pvc.yaml?raw=true ;\
	kubectl delete -f ./examples/templates/example-minio-data.yaml

keys-installation:
	@ ./release-tools/generate-keys.sh $(KEY_DIR) $(DATASET_OPERATOR_NAMESPACE);\
	export CA_PEM_B64="$$(openssl base64 -A < "$(KEY_DIR)/ca.crt")" ;\
	$(SHELL_EXPORT) kubectl -n $(DATASET_OPERATOR_NAMESPACE) create secret tls webhook-server-tls \
            --cert "$(KEY_DIR)/webhook-server-tls.crt" \
            --key "$(KEY_DIR)/webhook-server-tls.key" --dry-run -o yaml | kubectl apply -f - ;\
    $(SHELL_EXPORT) envsubst < "./src/dataset-operator/deploy/webhook.yaml.template" | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f -;\
	rm -rf $(KEY_DIR)
