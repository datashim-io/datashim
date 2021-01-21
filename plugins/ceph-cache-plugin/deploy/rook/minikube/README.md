# ROOK with CSI-LVM on minikube

## Tip 1: Minikube and losetup
```csi-lvm``` under ```MiniKube``` environment has some issues because of the losetup provided by busybox 
https://github.com/kubernetes/minikube/issues/8284. You need to transfer in your Minibox VM(s) a losetup 
from util-linux. We supply a helper script to automate this procedure (```./fix_minikube_losetup.py```). 
Note this script has been tested against K8S 1.19.4, MiniKube v1.16.0 with VirtualBox.

## Tip 2: Minikube multinode
Network of Minikube multinode deployments is experiencing weird issues. The only ```cni``` we found to work after 
the following tweak is ```calico``` so make sure to create your multinode minikube cluster with ```--cni=calico```

**Tweak:** After all nodes of the minikube multinode cluster are ready and all pods are in state running, we need to
delete the ```coredns-...-...``` pod. Afterward, k8s will create another ```coredns-...-...``` pod and the network of 
our minikube multinode cluster will be restored.



