BASE_EXTERNAL_PROVISIONER_IMAGE := csi-provisioner
EXTERNAL_PROVISIONER_IMAGE := $(DOCKER_REGISTRY)/$(BASE_EXTERNAL_PROVISIONER_IMAGE)
EXTERNAL_PROVISIONER_IMAGE := $(EXTERNAL_PROVISIONER_IMAGE):$(EXTERNAL_PROVISIONER_TAG)

BASE_EXTERNAL_ATTACHER_IMAGE := csi-attacher
EXTERNAL_ATTACHER_IMAGE := $(DOCKER_REGISTRY)/$(BASE_EXTERNAL_ATTACHER_IMAGE)
EXTERNAL_ATTACHER_IMAGE := $(EXTERNAL_ATTACHER_IMAGE):$(EXTERNAL_ATTACHER_TAG)

BASE_NODE_DRIVER_REGISTRAR_IMAGE := csi-node-driver-registrar
NODE_DRIVER_REGISTRAR_IMAGE := $(DOCKER_REGISTRY)/$(BASE_NODE_DRIVER_REGISTRAR_IMAGE)
NODE_DRIVER_REGISTRAR_IMAGE := $(NODE_DRIVER_REGISTRAR_IMAGE):$(NODE_DRIVER_REGISTRAR_TAG)

DATASET_OPERATOR_IMAGE := dataset-operator
DATASET_OPERATOR_TAG := archive
DATASET_OPERATOR_IMAGE := $(DOCKER_REGISTRY)/$(DATASET_OPERATOR_IMAGE)
DATASET_OPERATOR_IMAGE := $(DATASET_OPERATOR_IMAGE):$(DATASET_OPERATOR_TAG)

CSI_S3_IMAGE := csi-s3
CSI_S3_IMAGE_TAG := archive
CSI_S3_IMAGE := $(DOCKER_REGISTRY)/$(CSI_S3_IMAGE)
CSI_S3_IMAGE := $(CSI_S3_IMAGE):$(CSI_S3_IMAGE_TAG)

CSI_NFS_IMAGE := csi-nfs
CSI_NFS_IMAGE_TAG := latest
CSI_NFS_IMAGE := $(DOCKER_REGISTRY)/$(CSI_NFS_IMAGE)
CSI_NFS_IMAGE := $(CSI_NFS_IMAGE):$(CSI_NFS_IMAGE_TAG)

GENERAGE_KEYS_IMAGE := generate-keys
GENERAGE_KEYS_IMAGE_TAG := latest
GENERAGE_KEYS_IMAGE := $(DOCKER_REGISTRY)/$(GENERAGE_KEYS_IMAGE)
GENERAGE_KEYS_IMAGE := $(GENERAGE_KEYS_IMAGE):$(GENERAGE_KEYS_IMAGE_TAG)

#1: git repo url
#2: git tag
#3: directory created from pull
#4: image to be created
define install_sidecar
	@mkdir -p ./_tmp ;\
	if [ ! -d ./_tmp/$(3) ] ;\
    then \
    	  git clone $(1) ./_tmp/$(3); \
    fi
	cd ./_tmp/$(3) ;\
	git checkout $(2)
	cp ./Dockerfile ./_tmp/$(3)/Dockerfile-$(3).installer
	printf "\nRUN mkdir /tmp-code\
	\nCOPY go.mod /tmp-code/go.mod\
	\nCOPY go.sum /tmp-code/go.sum\
	\nRUN cd /tmp-code && go mod download\
	\nCOPY . /$(3)\
	\nRUN cd /$(3) && make build" | tee -a ./_tmp/$(3)/Dockerfile-$(3).installer ;\
	docker build --build-arg=ARCH=$(ARCH) -t $(3)-installer -f ./_tmp/$(3)/Dockerfile-$(3).installer ./_tmp/$(3) ;\
	mkdir -p ./_tmp/$(3)/bin ;\
	docker run --rm -v $$(pwd)/_tmp/$(3)/bin:/tmp-bin $(3)-installer sh -c "cp -r /$(3)/bin/* /tmp-bin" ;\
	cd ./_tmp/$(3) &&	docker build -t $(4)-$(ARCH) .
endef

#1: local directory name
#2: image to be created
define install_local
	cd ../src/$(1) && make container -e IMAGE_TAG=$(2)-$(ARCH) -e ARCH=$(ARCH)
endef
