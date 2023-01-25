container:
	docker buildx create --use
	docker buildx build --platform linux/amd64,linux/arm64,linux/ppc64le -t $(IMAGE_TAG) .
