module github.com/ctrox/csi-s3

require (
	github.com/container-storage-interface/spec v1.6.0
	github.com/golang/glog v1.2.4
	github.com/kubernetes-csi/csi-lib-utils v0.11.0 // indirect
	github.com/kubernetes-csi/csi-test v2.0.0+incompatible
	github.com/kubernetes-csi/drivers v1.0.2
	github.com/minio/minio-go v0.0.0-20190430232750-10b3660b8f09
	github.com/mitchellh/go-ps v0.0.0-20170309133038-4fdf99ab2936
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	golang.org/x/net v0.23.0
	google.golang.org/grpc v1.56.3
	k8s.io/mount-utils v0.23.0
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
)

go 1.15
