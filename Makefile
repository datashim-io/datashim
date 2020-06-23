DOCKER_REGISTRY_COMPONENTS ?= yiannisgkoufas
DOCKER_REGISTRY_SECRET ?= your_already_installed_secrets
# you need something like this:
# kubectl create secret generic regcred -n NAMESPACE_OF_OPERATOR --from-file=.dockerconfigjson=$(echo $HOME)/.docker/config.json --type=kubernetes.io/dockerconfigjson
PULL_COMPONENTS ?= true

DOCKER_REGISTRY_SIDECARS ?= quay.io/k8scsi
# if you pull from public use quay.io/k8scsi
PULL_SIDECARS ?= true

DATASET_OPERATOR_NAMESPACE ?= default
NAMESPACES_TO_MONITOR ?= default

# if you are building use master, if you are pulling use canary
EXTERNAL_ATTACHER_TAG ?= v2.1.1
#working in ppc64le v2.1.1
EXTERNAL_PROVISIONER_TAG ?= v1.5.0
#working in ppc64le v1.5.0
NODE_DRIVER_REGISTRAR_TAG ?= canary
#working in ppc64le master

minikube-uninstall: undeployment

undeployment:
	@for file in $(K8S_FILES); do \
        echo deleting $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
    done
	@for namespace in $(NAMESPACES_TO_MONITOR); do \
    	kubectl label namespace $$namespace monitor-pods-datasets- ;\
    done
	@kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) secret webhook-server-tls ;\
	$(SHELL_EXPORT) envsubst < ./src/dataset-operator/deploy/webhook.yaml.template | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f -

deployment: keys-installation
	@for file in $(K8S_FILES); do \
  		echo creating $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
	done
	@for namespace in $(NAMESPACES_TO_MONITOR); do \
		kubectl label namespace $$namespace monitor-pods-datasets=enabled ;\
	done

minikube-install: minikube-load-containers keys-installation deployment

kubernetes-install: build-containers push-containers keys-installation deployment

include release-tools/Makefile
