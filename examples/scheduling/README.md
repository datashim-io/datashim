# Cache placement aware scheduler using Datasets

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

(These instructions have been tested on `minikube v1.25.1` using the `virtualbox` driver and Kubernetes version v1.21.0 on MacOS Monterey. They will not work with Kubernetes v1.22 and the docker driver.)

### Satisfy pre-requisites
First, we start the minikube cluster

`$ minikube start --nodes 2 -p scheduler-test --kubernetes-version=v1.21.0 --driver=virtualbox `

Add additional hard disk of 10 GB to each of the nodes (appears as `/dev/sdb`) and reboot the VMs.

Next, we install Datashim

`$ cd $DATASHIM_HOME && make deployment`

Next, we have to install our object storage cache based on Ceph. Follow [this guide](https://github.com/datashim-io/datashim/blob/master/docs/Ceph-Caching.md#rookceph-installation) to install Rook-Ceph. Due to [this issue](https://github.com/datashim-io/datashim/issues/143), we'll have to generate and load the plugin operator image into minikube like so:

```
$ cd $DATASHIM_HOME/plugins/ceph-cache-plugin
$ make build-container
$ minikube image load quay.io/datashim/ceph-cache-plugin:latest-amd64
```
Open `$DATASHIM_HOME/plugins/ceph-cache-plugin/deploy/operator.yaml` and change line 24 `imagePullPolicy: Always` to `imagePullPolicy: IfNotPresent`. Then, in the `ceph-cache-plugin` directory, execute

```
$ make deployment
```

### Deploy the scheduler

(Optional) We tag the images that we have created for the plugin. Update the fields in `manifests/install/charts/as-a-second-scheduler/values.yaml` as necessary

```
$ docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest datashim-test/kube-scheduler:cacheaware
$ docker tag localhost:5000/scheduler-plugins/controller:latest datashim-test/controller:cacheaware
$ minikube image load datashim-test/kube-scheduler:cacheaware
$ minikube image load datashim-test/controller:cacheaware
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

## Testing the scheduler plugin

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

It'll take a while for the cache to create an instance of gateway and storage for a dataset. If successful, this is what you'll see:


```
$ kubectl describe dataset example-dataset
....
Status:
  Caching:
    Info:    Caching is assigned to ceph-cache-plugin plugin
    Status:  Pending
  Provision:
    Status:  OK
```
Ignore the `Status: Pending` message for caching. `Provision.Status` is reporting OK which means the cache has provided an instance

The internal dataset representation now reflects the placement information coming from our Ceph-cache

```
$ kubectl describe datasetinternal example-dataset
...
Status:
  Caching:
    Placements:
      Datalocations:
        Key:    kubernetes.io/hostname
        Value:  scheduler-test-m02
      Gateways:
        Key:    kubernetes.io/hostname
        Value:  scheduler-test-m02
```

Let's create a pod and assign it to our deployed scheduler for allocation

```
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: testpod
  labels:
    dataset.0.id: "example-dataset"
    dataset.0.useas: "mount"
spec:
  schedulerName: scheduler-plugins-scheduler
  containers:
    - name: nginx
      image: nginx
EOF
```

This pod gets scheduled to the node with the gateway for the cached dataset

```
$ kubectl get pods -o wide
NAME      READY   STATUS    RESTARTS   AGE     IP            NODE                 NOMINATED NODE   READINESS GATES
testpod   1/1     Running   0          2m36s   10.244.1.27   scheduler-test-m02   <none>           <none>
```

This can be verified from the logs of the scheduler pod

```
$ kubectl logs -n scheduler-plugins scheduler-plugins-scheduler-67b979c8db-ml6lm
....
I0203 12:55:53.636267       1 common.go:147] dataset example-dataset has Caching Status Pending because of Info Caching is assigned to ceph-cache-plugin plugin
I0203 12:55:53.636277       1 common.go:164] dataset example-dataset is cached.. fetching deployment information
I0203 12:55:53.636282       1 common.go:171] Gateway list for dataset example-dataset is [scheduler-test-m02]
I0203 12:55:53.636292       1 common.go:185] Data locations list for dataset example-dataset is [scheduler-test-m02]
I0203 12:55:53.636300       1 scoring.go:102] Node scheduler-test-m02 is sames as cache gateway scheduler-test-m02 for task pod testpod
...
```

if the gateway is overloaded, the plugin will favour other nodes in the same topology (zone followed by region). This, however, needs a larger cluster in a cloud provider
