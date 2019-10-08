SHELL=/bin/bash

CSI_S3_IMAGE := csi-s3
CSI_S3_IMAGE_TAG := dev

DATASET_OPERATOR_IMAGE := dataset-operator
DATASET_OPERATOR_TAG := latest

MAKE_ENV += EXTERNAL_PROVISIONER_IMAGE EXTERNAL_PROVISIONER_TAG
MAKE_ENV += EXTERNAL_ATTACHER_IMAGE EXTERNAL_ATTACHER_TAG
MAKE_ENV += NODE_DRIVER_REGISTRAR_IMAGE NODE_DRIVER_REGISTRAR_TAG
MAKE_ENV += CSI_S3_IMAGE CSI_S3_IMAGE_TAG
MAKE_ENV += DATASET_OPERATOR_IMAGE DATASET_OPERATOR_TAG
MAKE_ENV += DATASET_OPERATOR_NAMESPACE

SHELL_EXPORT := $(foreach v,$(MAKE_ENV),$(v)='$($(v))' )

K8S_FILES += $(shell find ./src/csi-s3/deploy/kubernetes -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/dataset-operator/deploy -maxdepth 1 -name '*.yaml')
K8S_FILES += $(shell find ./src/dataset-operator/deploy/crds -maxdepth 1 -name '*crd.yaml')

base:
	docker build -t docker-go-base --build-arg ARCH=$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; else echo "$$(arch)"; fi) --build-arg DOCKER_ARCH=$(shell if [ "$$(arch)" == "ppc64le" ]; then echo ppc64el;elif [ "$$(arch)" == "x86_64" ]; then echo amd64; fi) -f ./release-tools/docker-go-base/Dockerfile .

define install_sidecar
	@if [ ! -d _tmp/$(2) ] ;\
    then \
    	  git clone $(1) _tmp/$(2); \
    fi
	cd _tmp/$(2) ;\
	git checkout $(3)
	printf 'FROM docker-go-base\nCOPY ./$(2) /$(2)  \nWORKDIR /$(2)' > _tmp/Dockerfile.$(2)
	docker build -t $(2)-installer -f ./_tmp/Dockerfile.$(2) ./_tmp
	docker run -it -v /var/run/docker.sock:/var/run/docker.sock $(2)-installer make container -e IMAGE_TAG=$(2):$(3)
	rm _tmp/Dockerfile.$(2)
	rm -rf _tmp/$(2)
endef

define install_local
	cp -r ./src/$(1) _tmp
	printf 'FROM docker-go-base\nCOPY ./$(1) /$(1)  \nWORKDIR /$(1)' > _tmp/Dockerfile.$(1)
	docker build -t $(1)-installer -f ./_tmp/Dockerfile.$(1) ./_tmp
	docker run -it -v /var/run/docker.sock:/var/run/docker.sock $(1)-installer make container -e IMAGE_TAG=$(1):$(2)
	rm _tmp/Dockerfile.$(1)
	rm -rf _tmp/$(1)
endef

define load_containers_minikube
    @export docker_image=$(1)_$(2).tar.gz;\
    docker save $(1):$(2) | gzip > _tmp/$${docker_image////_};\
    eval $$(minikube docker-env) ;\
    docker load < _tmp/$${docker_image////_} ;\
    rm -rf _tmp/$${docker_image////_}
endef

build-containers:
ifeq ($(USE_IMAGES_FOR_SIDECARS), true)
	@docker pull $(EXTERNAL_PROVISIONER_IMAGE):$(EXTERNAL_PROVISIONER_TAG) ;\
	docker pull $(EXTERNAL_ATTACHER_IMAGE):$(EXTERNAL_ATTACHER_TAG) ;\
	docker pull $(NODE_DRIVER_REGISTRAR_IMAGE):$(NODE_DRIVER_REGISTRAR_TAG)
else
	$(call install_sidecar,https://github.com/kubernetes-csi/external-attacher.git,$(EXTERNAL_ATTACHER_IMAGE),$(EXTERNAL_ATTACHER_TAG))
	$(call install_sidecar,https://github.com/kubernetes-csi/external-provisioner.git,$(EXTERNAL_PROVISIONER_IMAGE),$(EXTERNAL_PROVISIONER_TAG))
	$(call install_sidecar,https://github.com/kubernetes-csi/node-driver-registrar.git,$(NODE_DRIVER_REGISTRAR_IMAGE),$(NODE_DRIVER_REGISTRAR_TAG))
endif
	$(call install_local,$(CSI_S3_IMAGE),$(CSI_S3_IMAGE_TAG))
	$(call install_local,$(DATASET_OPERATOR_IMAGE),$(DATASET_OPERATOR_TAG))

minikube-load-containers: build-containers
	$(call load_containers_minikube,$(EXTERNAL_ATTACHER_IMAGE),$(EXTERNAL_ATTACHER_TAG))
	$(call load_containers_minikube,$(EXTERNAL_PROVISIONER_IMAGE),$(EXTERNAL_PROVISIONER_TAG))
	$(call load_containers_minikube,$(NODE_DRIVER_REGISTRAR_IMAGE),$(NODE_DRIVER_REGISTRAR_TAG))
	$(call load_containers_minikube,$(CSI_S3_IMAGE),$(CSI_S3_IMAGE_TAG)-full)
	$(call load_containers_minikube,$(DATASET_OPERATOR_IMAGE),$(DATASET_OPERATOR_TAG))

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
	kubectl -n $(DATASET_OPERATOR_NAMESPACE) create secret tls webhook-server-tls \
            --cert "$${keydir}/webhook-server-tls.crt" \
            --key "$${keydir}/webhook-server-tls.key" --dry-run -o yaml | kubectl apply -f - ;\
    $(SHELL_EXPORT) envsubst < "./src/dataset-operator/deploy/webhook.yaml.template" | kubectl apply -f -;\
	rm -rf "$$keydir"

deployment:
	@for file in $(K8S_FILES); do \
		$(SHELL_EXPORT) envsubst < $$file | kubectl apply -f - ;\
	done
	@for namespace in $(NAMESPACES_TO_MONITOR); do \
		kubectl label namespace $$namespace monitor-pods-datasets=enabled ;\
	done

minikube: base minio-install minikube-load-containers keys-installation deployment