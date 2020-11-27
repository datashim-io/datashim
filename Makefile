DATASET_OPERATOR_NAMESPACE=dlf

manifests:
	helm template --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf.yaml
	helm template --set global.sidecars.kubeletPath="/var/data/kubelet" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-ibm-k8s.yaml
	helm template --set global.type="oc" --set global.sidecars.kubeletPath="/var/data/kubelet" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-ibm-oc.yaml
	helm template --set global.type="oc" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-oc.yaml
undeployment:
	kubectl delete -f ./release-tools/manifests/dlf.yaml
	kubectl label namespace default monitor-pods-datasets-
deployment:
	kubectl apply -f ./release-tools/manifests/dlf.yaml
	kubectl label namespace default monitor-pods-datasets=enabled
