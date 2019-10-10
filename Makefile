DOCKER_REGISTRY_COMPONENTS := yiannisgkoufas
DOCKER_REGISTRY_SECRET := regcred
# you need something like this:
# kubectl create secret generic regcred -n NAMESPACE_OF_OPERATOR --from-file=.dockerconfigjson=$(echo $HOME)/.docker/config.json --type=kubernetes.io/dockerconfigjson
PULL_COMPONENTS := false

DOCKER_REGISTRY_SIDECARS := quay.io/k8scsi
PULL_SIDECARS := true

DATASET_OPERATOR_NAMESPACE := default
NAMESPACES_TO_MONITOR := default

# if you are building use master, if you are pulling use latest
EXTERNAL_ATTACHER_TAG := v1.2.1
#working in ppc64le v1.2.1
EXTERNAL_PROVISIONER_TAG := v1.4.0-rc1
#working in ppc64le v1.4.0-rc1
NODE_DRIVER_REGISTRAR_TAG := v1.1.0
#working in ppc64le v1.1.0

include release-tools/Makefile