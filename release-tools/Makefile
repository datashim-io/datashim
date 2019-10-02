base:
	docker build -t docker-go-base --build-arg ARCH=$(shell if [ "$$(arch)" == "x86_64" ]; then echo amd64; else echo "$$(arch)"; fi) --build-arg DOCKER_ARCH=$(shell if [ "$$(arch)" == "ppc64le" ]; then echo ppc64el;elif [ "$$(arch)" == "x86_64" ]; then echo amd64; fi) -f ./release-tools/docker-go-base/Dockerfile .

define install_sidecar
	@if [ ! -d _tmp/$(3) ] ;\
    then \
    	  git clone $(1) _tmp/$(3); \
    fi
	cd _tmp/$(3) ;\
	git checkout $(2)
	printf 'FROM docker-go-base\nCOPY ./$(3) /$(3)  \nWORKDIR /$(3)' > _tmp/Dockerfile.$(3)
	docker build -t $(3)-installer -f ./_tmp/Dockerfile.$(3) ./_tmp
	docker run -it -v /var/run/docker.sock:/var/run/docker.sock $(3)-installer make container
	docker tag $(4) $(3):$(2)
	rm _tmp/Dockerfile.$(3)
	rm -rf _tmp/$(3)
endef

sidecar-containers: base
	$(call install_sidecar,https://github.com/kubernetes-csi/external-provisioner.git,$(EXTERNAL_PROVISIONER_TAG),external-provisioner,$(ORIGINAL_PROVISIONER_IMAGE))
	$(call install_sidecar,https://github.com/kubernetes-csi/node-driver-registrar.git,$(NODE_DRIVER_REGISTRAR_TAG),node-driver-registrar,$(ORIGINAL_NODE_DRIVER_REGISTRAR_IMAGE))

build: sidecar-containers