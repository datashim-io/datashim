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
#In case you want to monitor multiple namespaces use comma delimited
#NAMESPACES_TO_MONITOR ?= default,one-namespace,another-namespace

KUBELET_PATH ?= /var/lib/kubelet

# confirmed to be working in:
# -- openshift 4.4, x86_64
# -- minikube 1.12, x86_64
# -- kubernetes 1.18, x86_64
EXTERNAL_ATTACHER_TAG ?= v3.0.0-rc1
EXTERNAL_PROVISIONER_TAG ?= v2.0.0-rc2
NODE_DRIVER_REGISTRAR_TAG ?= v1.3.0

minikube-uninstall: undeployment

undeployment:
	@for file in $(K8S_FILES); do \
        echo deleting $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
    done
	@IFS=',' read -ra namespace_array <<< $(NAMESPACES_TO_MONITOR) &&\
	for namespace in "$${namespace_array[@]}"; \
	do\
    	kubectl label namespace $$namespace monitor-pods-datasets- ;\
    done
	@kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) secret webhook-server-tls

undeployment-operator-only:
	@for file in $(OPERATOR_ONLY_K8S_FILES); do \
        echo deleting $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
    done
	@IFS=',' read -ra namespace_array <<< $(DATASET_OPERATOR_NAMESPACE) &&\
	for namespace in "$${namespace_array[@]}"; \
	do\
    	kubectl label namespace $$namespace monitor-pods-datasets- ;\
    done
	@kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) secret webhook-server-tls ;\
	$(SHELL_EXPORT) envsubst < ./src/dataset-operator/deploy/webhook.yaml.template | kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) -f -

deployment:
	@for file in $(K8S_FILES); do \
  		echo creating $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
	done
	@IFS=',' read -ra namespace_array <<< $(NAMESPACES_TO_MONITOR) &&\
	for namespace in "$${namespace_array[@]}"; \
	do\
		kubectl label namespace $$namespace monitor-pods-datasets=enabled ;\
	done

deployment-operator-only:
	@for file in $(OPERATOR_ONLY_K8S_FILES); do \
  		echo creating $$file ;\
		$(SHELL_EXPORT) envsubst < $$file | kubectl apply -n $(DATASET_OPERATOR_NAMESPACE) -f - ;\
	done
	@IFS=',' read -ra namespace_array <<< $(NAMESPACES_TO_MONITOR) &&\
	for namespace in "$${namespace_array[@]}"; \
	do\
		kubectl label namespace $$namespace monitor-pods-datasets=enabled ;\
	done



minikube-install: minikube-load-containers keys-installation deployment

kubernetes-install: build-containers push-containers keys-installation deployment

include release-tools/Makefile
