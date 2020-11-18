**The instructions currently work only for the branch https://github.com/IBM/dataset-lifecycle-framework/tree/fixed-caching**

# Minikube installation

First we need to have a working cluster.

`minikube start --memory='10G' --cpus=8 --disk-size='15g' --driver=docker`

# Rook/Ceph Installation

We need to have ceph installed. Inside `plugins/ceph-cache-plugin/deploy/rook` directory execute:
``` bash
kubectl create -f common.yaml
```
Inspect the file `keys-installation.sh` and make sure you replace YOUR_REGISTRY,YOUR_EMAIL,YOUR_PASSWORD with the correct values for your docker registry and execute:
``` bash
./keys-installation.sh
```
Inspect the file operator.yaml and replace the value `YOUR_REGISTRY` and execute:
``` bash
kubectl create -f operator.yaml
```
Inspect the file cluster.yaml and replace the value `YOUR_REGISTRY` and execute:
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

# DLF Installation

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
apiVersion: com.ie.ibm.hpsys/v1alpha1
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

# Ceph Caching Plugin Installation

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