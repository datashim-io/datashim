# Using Ceph for caching
## Installation
### Method 1 (Recommended)

Inside `plugins/ceph-cache-plugin/deploy/rook` directory execute:
``` bash
kubectl create -f common.yaml
kubectl create -f operator.yaml
```
Inspect the file cluster.yaml and setup according to the nodes and the dedicated ceph-wise disk devices, the value of `storage.nodes` e.g.
```yaml
storage:
    useAllNodes: false
    useAllDevices: false
    nodes:
      - name: "minikube"
        devices: 
          - name: "sdb"
            config:
              storeType: bluestore
              osdsPerDevice: "1"
```
Afterward, execute:
``` bash
kubectl create -f cluster.yaml
```
If everything worked correctly the pods in the rook-ceph namespace should be like this:
``` bash
rook-ceph-mgr-a-5f8f5c978-xgcpw                     1/1     Running     0          79s
rook-ceph-mon-a-6879b87656-bxbrw                    1/1     Running     0          89s
rook-ceph-operator-86f9b59b8-2fvkb                  1/1     Running     0          5m29s
rook-ceph-osd-0-9dcb46c48-hrzvz                     1/1     Running     0          43s
```
**NOTE** If you want to delete/create a new cluster, besides invoking `kubectl delete -f cluster.yaml`
You need also to delete the paths in defined in `dataDirHostPath` and directories.path

Now we can proceed with installing DLF.

### Method 2 (Testing)

If you are after maximum performance we strongly advice to set up your ceph cluster according to the method above. However, for testing purposes and/or lacking of disk devices we describe a method to test this inside minikube and provide a script `plugins/ceph-cache-plugin/deploy/rook/setup_ceph_cluster.sh` that installs rook with csi-lvm storage class. 


#### Minikube installation

First we need to have a working cluster.

`minikube start --memory='6G' --cpus=4 --disk-size='40g' --driver=virtualbox -p rooktest`

**NOTE:** run ```./minikube/fix_minikube_losetup.py``` to bypass the [current issue](https://github.com/kubernetes/minikube/issues/8284) of minikube with loset.

**NOTE2:** if you change the disk-size of the minikube command make sure to tune accordingly the following parameters


#### CSI-LVM setup

Before invoking the script you should tune according to your needs the following attributes

| Attribute | File | Description |
|---|---|---|
GIGA_SPACE | `plugins/ceph-cache-plugin/deploy/rook/csi-lvm-setup/create-loops.yaml` | Size of the loop device that csi-lvm will create on each node |
`spec.mon.volumeClaimTemplate.spec.resources.requests.storage` | `plugins/ceph-cache-plugin/deploy/rook/cluster-on-pvc.yaml` | Storage Size of mon ceph service |Size of the loop device that csi-lvm will create on each node |
`spec.storage.storageClassDeviceSets.volumeClaimTemplates.spec.resources.requests.storage` | `plugins/ceph-cache-plugin/deploy/rook/cluster-on-pvc.yaml` | Storage size of CEPH osds |
`spec.storage.storageClassDeviceSets.count` | `plugins/ceph-cache-plugin/deploy/rook/cluster-on-pvc.yaml` | Total number of CEPH osds |

The command line arguments of the script are the names of the nodes that the csi-lvm should create loop devices on and the corresponding CEPH services will run on, e.g.

```bash
cd plugins/ceph-cache-plugin/deploy/rook && \
./setup_ceph_cluster.sh nodename1 ...
```

Keep in mind that the script will uninstall any previous installations of csi-lvm and rook-ceph which made through the script. If no command line arguments are passed to the script this will result in uninstalling everything.
## DLF Installation

Go into the root of this directory and execute:
`make deployment`

The pods in the default namespace would look like this:
``` bash
csi-attacher-nfsplugin-0            2/2     Running   0          7s
csi-attacher-s3-0                   1/1     Running   0          8s
csi-nodeplugin-nfsplugin-nqgtl      2/2     Running   0          7s
csi-provisioner-s3-0                2/2     Running   0          8s
csi-s3-k9b5j                        2/2     Running   0          8s
dataset-operator-7b8f65f7d4-hg8n5   1/1     Running   0          6s
```
Create an s3 dataset by replacing the values and invoking `kubectl create -f my-dataset.yaml`
``` yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: example-dataset
spec:
  local:
    type: "COS"
    accessKeyID: "{AWS_ACCESS_KEY_ID}"
    secretAccessKey: "{AWS_SECRET_ACCESS_KEY}"
    endpoint: "{S3_SERVICE_URL}"
    bucket: "{BUCKET_NAME}"
    region: "" #it can be empty
```

Now if you check about datasetsinternal and PVC you would be able to see the example-dataset
``` bash
kubectl get datasetsinternal
kubectl get pvc
```
Delete the dataset we created before by executing `kubectl delete dataset/example-dataset`
If you execute `kubectl describe datasetinternal/example-dataset` you would see the credentials and the endpoints you originally specified.

Let's try to add the caching plugin.

## Ceph Caching Plugin Installation

Change into the directory and invoke:
`make deployment`

Let's create the same dataset now that the plugin is deployed:
`kubectl create -f my-dataset.yaml`

You should see a new rgw pod starting up on rook-ceph namespace:
``` bash
rook-ceph-rgw-test-a-77f78b7b69-z5kp9              1/1     Running     0          4m43s
```
After a couple of minutes if you list datasetsinternal you will see the example-dataset created. 
If you describe it using `kubectl describe datasetinternal/example-dataset` you will notice that the credentials are different and they point to the rados gateway instance, therefore the PVC would reflect the cached version of the dataset.