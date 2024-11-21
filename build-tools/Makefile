DOCKER_REGISTRY ?= quay.io/datashim-io

ARCH ?= $(shell go env GOARCH)

#used only when building the plugins

EXTERNAL_ATTACHER_TAG ?= v4.7.0
EXTERNAL_PROVISIONER_TAG ?= v5.1.0
NODE_DRIVER_REGISTRAR_TAG ?= v2.12.0

include helpers.mk

build-sidecars:
	$(call install_sidecar,https://github.com/kubernetes-csi/external-attacher.git,$(EXTERNAL_ATTACHER_TAG),external-attacher,$(EXTERNAL_ATTACHER_IMAGE),$(BASE_EXTERNAL_ATTACHER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/external-provisioner.git,$(EXTERNAL_PROVISIONER_TAG),external-provisioner,$(EXTERNAL_PROVISIONER_IMAGE),$(BASE_EXTERNAL_PROVISIONER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/node-driver-registrar.git,$(NODE_DRIVER_REGISTRAR_TAG),node-driver-registrar,$(NODE_DRIVER_REGISTRAR_IMAGE),$(BASE_NODE_DRIVER_REGISTRAR_IMAGE))

build-components:
	$(call install_local,dataset-operator,$(DATASET_OPERATOR_IMAGE))
	$(call install_local,generate-keys,$(GENERATE_KEYS_IMAGE))

build-csi-plugins:
	$(call install_local,csi-s3,$(CSI_S3_IMAGE))
	$(call install_local,csi-driver-nfs,$(CSI_NFS_IMAGE))

push-sidecars:
	@docker login -u=${DOCKER_USER} -p=${DOCKER_PASSWORD} quay.io ;\
	docker push $(EXTERNAL_PROVISIONER_IMAGE)-$(ARCH) ;\
	docker push $(EXTERNAL_ATTACHER_IMAGE)-$(ARCH) ;\
	docker push $(NODE_DRIVER_REGISTRAR_IMAGE)-$(ARCH)

push-components:
	@docker login -u=${DOCKER_USER} -p=${DOCKER_PASSWORD} quay.io ;\
	docker push $(DATASET_OPERATOR_IMAGE)-$(ARCH) ;\
	docker push $(GENERATE_KEYS_IMAGE)-$(ARCH)

push-csi-plugins:
	@docker login -u=${DOCKER_USER} -p=${DOCKER_PASSWORD} quay.io ;\
	docker push $(CSI_S3_IMAGE)-$(ARCH) ;\
	docker push $(CSI_NFS_IMAGE)-$(ARCH)

push-multiarch-core:
	$(call generate_push_multi_arch_manifest,$(DATASET_OPERATOR_IMAGE))
	$(call generate_push_multi_arch_manifest,$(GENERATE_KEYS_IMAGE))

push-multiarch-csi-plugins:
	$(call generate_push_multi_arch_manifest,$(CSI_S3_IMAGE))
	$(call generate_push_multi_arch_manifest,$(CSI_NFS_IMAGE))
