DATASET_OPERATOR_NAMESPACE=dlf

COMMON_IMAGE_TAG ?= latest

HELM_IMAGE_TAGS = --set global.namespaceYaml="true" --set csi-nfs-chart.csinfs.tag=$(COMMON_IMAGE_TAG) --set csi-s3-chart.csis3.tag=$(COMMON_IMAGE_TAG) --set dataset-operator-chart.generatekeys.tag=$(COMMON_IMAGE_TAG) --set dataset-operator-chart.datasetoperator.tag=$(COMMON_IMAGE_TAG)

manifests:
	helm template $(HELM_IMAGE_TAGS) --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default --set csi-h3-chart.enabled="false" chart/ > release-tools/manifests/dlf.yaml
	helm template $(HELM_IMAGE_TAGS) --set global.sidecars.kubeletPath="/var/data/kubelet" --set global.sidecars.kubeletPath="/var/data/kubelet" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-ibm-k8s.yaml
	helm template $(HELM_IMAGE_TAGS) --set global.type="oc" --set csi-h3-chart.enabled="false" --set global.sidecars.kubeletPath="/var/data/kubelet" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-ibm-oc.yaml
	helm template $(HELM_IMAGE_TAGS) --set global.type="oc" --set csi-h3-chart.enabled="false" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-oc.yaml
undeployment:
	kubectl delete -f ./release-tools/manifests/dlf.yaml
	kubectl label namespace default monitor-pods-datasets-
deployment:
	kubectl apply -f ./release-tools/manifests/dlf.yaml
	kubectl label namespace default monitor-pods-datasets=enabled
