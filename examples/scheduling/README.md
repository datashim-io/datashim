### Cache placement aware scheduler using Datasets

This is an example of a Kubernetes scheduler plugin that uses the placement information exposed by Datashim for cached datasets to schedule pods close to the gateway (if the cache is a service that follows a protocol like S3), or on the nodes that have the dataset (if the cache is a distributed file system). It is a fork of the [out-of-tree scheduler plugins](https://github.com/kubernetes-sigs/scheduler-plugins) repository and is built based on its instructions. The pre-requisites to get this plugin are:

1. [Datashim](https://github.com/datashim-io/datashim) is installed in the cluster
2. A cache implementation that exposes the placement information of the datasets (e.g. our [Ceph-based object storage cache](https://github.com/datashim-io/datashim/tree/master/plugins/ceph-cache-plugin)) is installed and configured in the cluster as well.

Without the cache, the scheduler will work but will not be able to use any placement information. Below are the steps to checkout the scheduler, and get it working in minikube.

## Get the code

For the purposes of this exercise, we will designate the root directory of Datashim as `DATASHIM_HOME`. For example, if Datashim has been cloned into `$HOME/go/src/github.com/datashim-io/datashim`,then that is now `DATASHIM_HOME`. Go to `$DATASHIM_HOME/examples/scheduling/src/sigs.k8s.io/scheduler-plugins` like so:

`$ cd $DATASHIM_HOME/examples/scheduling/src/sigs.k8s.io/scheduler-plugins`

The scheduler code is included as a submodule so this directory should be currently empty. Fetch the scheduler plugin code by

```
$ git submodule init
Submodule 'examples/scheduling/src/sigs.k8s.io/scheduler-plugins' (https://github.com/srikumar003/scheduler-plugins.git) registered for path './'
$ ls
$ git submodule update
Cloning into '/private/tmp/datashim/examples/scheduling/src/sigs.k8s.io/scheduler-plugins'...
Submodule path './': checked out '5765d192558809274360121f17d49b7384554df1'
$ ls
ls
CONTRIBUTING.md                  OWNERS ...
```

## Compile the code

Override the GOPATH temporarily by

`$ export GOPATH=$DATASHIM_HOME/examples/scheduling/src/sigs.k8s.io/scheduler-plugins`

and then execute

`$ make local-image`

On completion, 2 new docker images should be present:

```
$ docker images
REPOSITORY                                        TAG          IMAGE ID       CREATED         SIZE
localhost:5000/scheduler-plugins/kube-scheduler   latest       0e5ff8eb09d2   4 minutes ago   53.7MB
localhost:5000/scheduler-plugins/controller       latest       f96bad181baf   21 hours ago    48.5MB
...
```

## Deploy the scheduler plugin

The scheduler should be installed in a cluster with the pre-requisites fulfilled. For illustrative purposes, we will deploy the scheduler plugin in a `minikube` cluster as a second scheduler.

We'll use Helm for this purpose. The key files involved in this process are `manifests/install/charts/as-a-second-scheduler/templates/deployment.yaml`, `manifests/install/charts/as-a-second-scheduler/values.yaml`, and `manifests/cacheaware/scheduler-config.yaml` 

First, we start the minikube cluster

`$ minikube start --nodes 2 -p datashim-test`

Next, we install Datashim

`$ cd $DATASHIM_HOME && make deployment`

(Optional) We tag the images that we have created for the plugin. Update the fields in `manifests/install/charts/as-a-second-scheduler/values.yaml` as necessary

```
$ docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest datashim-test/kube-scheduler:cacheaware
$ docker tag localhost:5000/scheduler-plugins/controller:latest datashim-test/controller:cacheaware
$ minikube load image datashim-test/kube-scheduler:cacheaware
$ minikube load image datashim-test/controller:cacheaware
```

Next, we install the scheduler using Helm

```
$ cd $DATASHIM_HOME/examples/scheduling/src/sigs.k8s.io/scheduler-plugins
$ helm install cacheaware manifests/install/charts/as-a-second-scheduler/
```

Verify that the scheduler plugin pods are running

```
$ kubectl get pods -n scheduler-plugins
NAME                                            READY   STATUS  
scheduler-plugins-controller-5d94d8cf9f-ghkfb   1/1     Running  
scheduler-plugins-scheduler-67b979c8db-ml6lm    1/1     Running   
```

### Testing the scheduler plugin

First lets create a dataset

```
$ cat <<EOF | kubectl apply -f -
---
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: example-dataset
spec:
  local:
    type: "COS"
    secret-name: "<secret-name>"
    endpoint: "<endpoint-name>"
    bucket: "<bucket-name>"
EOF
```

`$ kubectl describe dataset example-dataset`

