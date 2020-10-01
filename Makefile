DATASET_OPERATOR_NAMESPACE=dlf

manifests:
	helm template --namespace=$(DATASET_OPERATOR_NAMESPACE) --name-template=default chart/ > release-tools/manifests/dlf.yaml
undeployment:
	kubectl delete -n $(DATASET_OPERATOR_NAMESPACE) clusterrole,clusterrolebinding,ds,deploy,job,mutatingwebhookconfiguration,ns,role,rolebinding,svc,sa,statefulset,storageclass -l app.kubernetes.io/name=dlf
	#or maybe just kubectl delete -f ./release-tools/manifests/ ?
	kubectl label namespace default monitor-pods-datasets-
deployment:
	kubectl apply -f ./release-tools/manifests/
	kubectl label namespace default monitor-pods-datasets=enabled
