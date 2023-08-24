DATASET_OPERATOR_NAMESPACE=dlf

COMPONENT_TAG ?= latest
CSI_PLUGIN_TAG ?= latest

HELM_CSI_OPTIONS = --set csi-nfs-chart.csinfs.tag=$(CSI_PLUGIN_TAG) --set csi-s3-chart.csis3.tag=$(CSI_PLUGIN_TAG) 
HELM_OPERATOR_OPTIONS = --set dataset-operator-chart.generatekeys.tag=$(COMPONENT_TAG) --set dataset-operator-chart.datasetoperator.tag=$(COMPONENT_TAG)

manifests:
	helm template $(HELM_CSI_OPTIONS) $(HELM_OPERATOR_OPTIONS) --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default --set csi-h3-chart.enabled="false" chart/ > release-tools/manifests/dlf.yaml
	helm template $(HELM_CSI_OPTIONS) $(HELM_OPERATOR_OPTIONS) --set global.sidecars.kubeletPath="/var/data/kubelet" --set global.sidecars.kubeletPath="/var/data/kubelet" --set csi-h3-chart.enabled="false" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-ibm-k8s.yaml
	helm template $(HELM_CSI_OPTIONS) $(HELM_OPERATOR_OPTIONS) --set global.type="oc" --set csi-h3-chart.enabled="false" --set global.sidecars.kubeletPath="/var/data/kubelet" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-ibm-oc.yaml
	helm template $(HELM_CSI_OPTIONS) $(HELM_OPERATOR_OPTIONS) --set global.type="oc" --set csi-h3-chart.enabled="false" --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf-oc.yaml
undeployment:
	kubectl delete -f ./release-tools/manifests/dlf.yaml
	kubectl delete ns $(DATASET_OPERATOR_NAMESPACE)
	kubectl label namespace default monitor-pods-datasets-
deployment:
	kubectl create ns $(DATASET_OPERATOR_NAMESPACE)
	kubectl apply -f ./release-tools/manifests/dlf.yaml
	kubectl label namespace default monitor-pods-datasets=enabled
