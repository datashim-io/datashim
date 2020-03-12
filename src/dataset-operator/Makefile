container:
#	operator-sdk generate k8s
#	operator-sdk generate openapi
#	operator-sdk build $(IMAGE_TAG)
	docker build -t $(IMAGE_TAG) . --build-arg=ARCH=$(ARCH)