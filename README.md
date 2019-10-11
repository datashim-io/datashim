# Dataset Lifecycle Framework

*__Dataset Lifecycle Framework__* gives you hassle-free access to remote datasets inside your Kubernetes applications.
Its components run as Kubernetes pods and can be installed in any flavor of Kubernetes(v1.15+).

In order to leverage its capabilities we provide you a **Dataset CRD**(*Custom Resource Definition*) and you just need to
**annotate your pods** accordingly to make the Dataset available inside your application.

It's built on [Operator SDK](https://github.com/operator-framework/operator-sdk) and it's extensible to support any
[CSI](https://kubernetes-csi.github.io/docs/) enabled storage system.

## Quickstart

If you prefer to watch a quick demo of its functionality, have a look in the recording:
[Demo](https://asciinema.org/a/273767)

The following steps demonstrate how to quickly get started with our framework using minikube. Check the 
[Minikube documentation](https://kubernetes.io/docs/setup/learning-environment/minikube/)
for instructions about how to install it. In case you want to deploy our framework on a proper kubernetes
cluster inspect the [Makefile](Makefile) to tailor your Dataset Lifecycle Framework installation.

### Requirements
- Docker
- git
- Kubernetes CLI (*kubectl*)

### Deployment and usage
Before starting this step, please make sure your Kubernetes CLI (*kubectl*) is
properly configured to interact with your minikube environment. The command `make minikube-install` will
take a bit as it builds the framework's components from scratch.

```bash
$ make minikube-install
```

Verify the installation by making sure the following pods are running:
```
$ kubectl get pods
csi-attacher-s3-0                   1/1     Running     0          53m
csi-provisioner-s3-0                2/2     Running     0          53m
csi-s3-qwv7t                        2/2     Running     0          53m
dataset-operator-54b74d5885-bg7sw   1/1     Running     0          53m
```

As part of the minikube installation we deployed minio and added sample data for demo purposes.
As a user now you can use any Dataset stored on minio inside your pods. Execute the following:
```
$ export MINIO_SERVICE_URL=$(minikube service minio-service --url)
$ envsubst < ./examples/example-dataset.yaml | kubectl create -f -
$ kubectl create -f ./examples/example-pod.yaml
```

What happened with the above commands? First we retrieved the URL of minio inside minikube `minikube service minio-service --url`
If instead you are working with another Cloud Object Store, feel free to use it!

In the next command we replaced in the [example-dataset](./examples/example-dataset.yaml) the address of minio and created
the new Dataset object. 
Also we have filled out the demo credentials so you need to modify accordingly if using a Cloud Object Store.

<pre>
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: <b>example-dataset</b>
spec:
  local:
    type: "COS"
    accessKeyID: "minio"
    secretAccessKey: "minio123"
    endpoint: <b>"${MINIO_SERVICE_URL}"</b>
    bucket: "my-bucket"
    region: "" #it can be empty
</pre>

Now inspect the [example-pod](./examples/example-pod.yaml) to see how to use the newly created **example-dataset**

<pre>
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    dataset.0.id: <b>"example-dataset"</b>
    dataset.0.useas: "mount"
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts:
        - mountPath: "/mount/dataset1" #optional, if not specified it would be mounted in /mnt/datasets/example-dataset
          name: <b>"example-dataset"</b>
</pre>

Notice the way we annotate the pod to make it aware of the datasets. For instance if we wanted to use multiple datasets,
in the labels section we would have something like this:

```
dataset.0.id: dataset-0
dataset.0.useas: mount

dataset.1.id: dataset-1
dataset.1.useas: mount

dataset.2.id: dataset-2
dataset.2.useas: mount
```

The part below `volumeMounts` is optional and can be used if the user wants to mount each dataset in a specific location.
If the user doesn't specify the mount point, as a convention we will mount the dataset on `/mnt/datasets/example-dataset`.

## Next Steps

The current release only provides support for datasets stored as S3 buckets and
requires users/administrators to input the access information for the dataset. We
are working on enabling the capability of fetching the dataset information from
a central catalog to facilitate access to organization-wide data lakes. Stay
tuned for updates.

Moreover we will add support for NFS datasets and other storage systems. 