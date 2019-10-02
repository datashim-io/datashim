EXTERNAL_PROVISIONER_TAG := v1.4.0-rc1
NODE_DRIVER_REGISTRAR_TAG := v1.0.2

ORIGINAL_PROVISIONER_IMAGE := csi-provisioner:latest
ORIGINAL_NODE_DRIVER_REGISTRAR_IMAGE := quay.io/k8scsi/csi-node-driver-registrar:v1.0-canary

all: build

include release-tools/Makefile