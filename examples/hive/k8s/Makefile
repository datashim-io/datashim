SHELL=/bin/bash

DATASET_OPERATOR_NAMESPACE ?= default

DOCKER_REGISTRY_COMPONENTS ?= the_registry_to_use_for_components
DOCKER_REGISTRY_SECRET ?= your_already_installed_secrets

HIVE_FILE_PATH := $(shell pwd)

HIVE_BASE_DOCKERFILE := $(HIVE_FILE_PATH)/Dockerfile
HIVE_BASE_IMAGE := dlf-hive-base
HIVE_BASE_TAG := latest
HIVE_BASE_IMAGE := $(HIVE_BASE_IMAGE):$(HIVE_BASE_TAG)

HIVESERVER_IMAGE := hive-server
HIVESERVER_TAG := latest
HIVESERVER_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(HIVESERVER_IMAGE)
HIVESERVER_IMAGE := $(HIVESERVER_IMAGE):$(HIVESERVER_TAG)
HIVESERVER_DOCKERFILE := $(HIVE_FILE_PATH)/Dockerfile.hiveserver

HIVEMETASTORE_IMAGE := hive-metastore
HIVEMETASTORE_TAG := latest
HIVEMETASTORE_IMAGE := $(DOCKER_REGISTRY_COMPONENTS)/$(HIVEMETASTORE_IMAGE)
HIVEMETASTORE_IMAGE := $(HIVEMETASTORE_IMAGE):$(HIVEMETASTORE_TAG)
HIVEMETASTORE_DOCKERFILE := $(HIVE_FILE_PATH)/Dockerfile.metastore

MAKE_ENV += DATASET_OPERATOR_NAMESPACE
MAKE_ENV += DOCKER_REGISTRY_SECRET
MAKE_ENV += HIVESERVER_IMAGE
MAKE_ENV += HIVEMETASTORE_IMAGE

SHELL_EXPORT := $(foreach v,$(MAKE_ENV),$(v)='$($(v))' )

#K8S_FILES += $(shell find $(HIVE_FILE_PATH)/deploy -maxdepth 1 -name '*.yaml')
define load_containers_minikube
	$(info Loading container $(2) on minikube)
    @mkdir -p _tmp;\
    driver_check=$$(cat $$HOME/.minikube/machines/minikube/config.json  | grep DriverName);\
    eval $$(minikube docker-env);\
    if [[ $$driver_check != *"none"* ]]; then docker save $(2) | gzip > _tmp/$(1).tar.gz;\
    docker load < _tmp/$(1).tar.gz ;\
    eval $$(minikube docker-env);\
    rm -rf ./release-tools/_tmp/$(1).tar.gz
    docker load < _tmp/$(1).tar.gz;\
    eval $$(minikube docker-env -u);\
    rm -rf _tmp/$(1).tar.gz;\
    fi;
endef

define build-images
	$(info Building $(1))
	docker build -t $(1) -f $(2) $(3) 
endef

define deploy-k8s
	$(info Deploying $(1))
	@$(SHELL_EXPORT) envsubst < $(HIVE_FILE_PATH)/deploy/$(1).yaml | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f - 
	@sleep 30s; kubectl wait --namespace $(DATASET_OPERATOR_NAMESPACE) --for condition=ready pods -l app=$(1) --timeout=90s > /dev/null 2>&1
endef  

define undeploy-k8s
	$(info Undeploying $(1))
	@$(SHELL_EXPORT) envsubst < $(HIVE_FILE_PATH)/deploy/$(1).yaml | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) --ignore-not-found --wait -f - 
endef

noobaa-env:
ifeq ($(origin S3_ENDPOINT),undefined)
$(info Getting connection parameters from Noobaa)
ifeq ($(origin NOOBAA_HOME),environment)
NOOBAA_HOME := ${NOOBAA_HOME}
else 
$(error NOOBAA_HOME not found or unset)
endif
S3_ENDPOINT := $(shell minikube service s3 --url | head -n1)
AWS_ACCESS_KEY_ID := $(shell $(NOOBAA_HOME)/noobaa status 2>/dev/null | grep AWS_ACCESS_KEY_ID | awk -F ": " '{print $$2}')
AWS_SECRET_ACCESS_KEY := $(shell $(NOOBAA_HOME)/noobaa status 2>/dev/null | grep AWS_SECRET_ACCESS_KEY | awk -F ": " '{print $$2}')
else ifeq ($(origin S3_ENDPOINT),environment)
$(info Getting connection parameters from env)
S3_ENDPOINT := ${S3_ENDPOINT}
AWS_ACCESS_KEY_ID := ${AWS_ACCESS_KEY_ID}
AWS_SECRET_ACCESS_KEY := ${AWS_SECRET_ACCESS_KEY}
endif
	
conf/hive-site.xml: noobaa-env
	@sed -e "s|\$${S3_ENDPOINT}|$(S3_ENDPOINT)|g" conf/hive-site.tmpl > conf/hive-site.xml
	
conf/metastore-site.xml: noobaa-env
	@sed -e "s|\$${S3_ENDPOINT}|$(S3_ENDPOINT)|g" conf/metastore-site.tmpl > conf/metastore-site.xml

deploy/s3-secret.yaml: noobaa-env
	@sed -e  "s|\$${AWS_ACCESS_KEY_ID}|$(AWS_ACCESS_KEY_ID)|g" \
		-e  "s|\$${AWS_SECRET_ACCESS_KEY}|$(AWS_SECRET_ACCESS_KEY)|g" \
        deploy/s3-secret.tmpl > deploy/s3-secret.yaml

build-images: conf/hive-site.xml conf/metastore-site.xml 
	$(call build-images,$(HIVE_BASE_IMAGE),$(HIVE_BASE_DOCKERFILE),$(HIVE_FILE_PATH))
	$(call build-images,$(HIVEMETASTORE_IMAGE),$(HIVEMETASTORE_DOCKERFILE),$(HIVE_FILE_PATH))
	$(call build-images,$(HIVESERVER_IMAGE),$(HIVESERVER_DOCKERFILE),$(HIVE_FILE_PATH))

push-images: build-images
	@docker push $(HIVESERVER_IMAGE) ;\
	docker push $(HIVEMETASTORE_IMAGE)

minikube-load-containers: build-images
	$(call load_containers_minikube,$(HIVEMETASTORE_IMAGE),hivemetastore)
	$(call load_containers_minikube,$(HIVESERVER_IMAGE),hiveserver)

deploy-secret: deploy/s3-secret.yaml
	@kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f deploy/s3-secret.yaml; \
	 kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f deploy/database-secret.yaml

deploy-database: deploy-secret
	$(call deploy-k8s,database)

deploy-hivemetastore: minikube-load-containers deploy-database
	$(call deploy-k8s,hivemetastore)

deploy-hive: deploy-hivemetastore
	$(call deploy-k8s,hiveserver)

undeploy-hive:
	$(call undeploy-k8s,hiveserver)
	$(call undeploy-k8s,hivemetastore)
	$(call undeploy-k8s,database)
	$(call undeploy-k8s,database-secret)
	$(call undeploy-k8s,s3-secret)

minikube-install: deploy-hive

minikube-uninstall: undeploy-hive
