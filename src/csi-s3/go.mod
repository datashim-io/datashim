module github.com/ctrox/csi-s3

require (
	github.com/container-storage-interface/spec v1.9.0
	github.com/golang/glog v1.2.1
	github.com/kubernetes-csi/csi-test v2.0.0+incompatible
	github.com/kubernetes-csi/drivers v1.0.2
	github.com/minio/minio-go v0.0.0-20190430232750-10b3660b8f09
	github.com/mitchellh/go-ps v0.0.0-20170309133038-4fdf99ab2936
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	golang.org/x/net v0.26.0
	google.golang.org/grpc v1.65.0
	k8s.io/mount-utils v0.30.0
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8
)

require (
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kubernetes-csi/csi-lib-utils v0.19.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/nxadm/tail v1.4.4 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/ini.v1 v1.41.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
)

go 1.22.5

toolchain go1.23.2
