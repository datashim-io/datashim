# Dataset Lifecycle Framework

You have reached the home of the *__Dataset Lifecycle Framework__*, a Kubernetes integrated framework for hassle free handling of data sources.

It is extremely common these days to deal with multiple data sources (datasets) hosted by public cloud storage solutions (e.g., IBM Cloud Object Store) accessible via public endpoints and credentials. Users interact with many of these datasets that are potentially offered by different providers and hosted in different locations around the globe. Using those datasets often involves many manual steps: knowing the credentials, crafting your pods to be able of mounting/accessing the data, etc. Plus, organizations with a distributed infrastructure and many employees accessing data might want to control who is allowed of accessing each dataset, and enable sharing of datasets within teams. 

The Dataset Lifecycle Framework enables users or administrators of Kubernetes clusters to easily link applications with data sources. Thanks to the new Dataset CRD all you need to do is to create a dataset, and then just include the dataset ID in your pods specification. Yep, it is just as easy as it sounds. Our framework will take care of all the dirty details of mounting or giving your pods access to the data. Once a dataset exists in a cluster, users will just need to reference it using the unique ID used at creation time. No need to provide access information at any further use.  

We are currently supporting datasets hosted as buckets in cloud object storage exposing the S3 API, and working on extending our support to other cloud-friendly storage solutions. 

Continue reading to learn how to quickly deploy and try the Dataset Lifecycle Framework on your favourite Kubernetes installation. 

## Getting started
The foolowing steps demonstrate how to use our framework and rely on
Minikube, for quick setup. Check the [Minikube
documentation](https://kubernetes.io/docs/setup/learning-environment/minikube/)
to know how to install it. In case you want to install on a proper kubernetes
cluster click [here](link) to get a more complete overview of how to build and deploy the
Dataset Lifecycle Framework.

### Requirements
- Docker
- git
- Kubernetes CLI (*kubectl*)

### Build and deployment
Before starting this step, please make sure your Bubernetes CLI (*kubectl*) is
properly configured to interact with your minikube environment.
In this step all the components of the Dataset Lifecycle Framework are
built from source and deployed on the target cluster.

```bash
$ make minikube
```

This will take a few minutes while the build process compiles and deploys all
the components needed. The result of the build process is a set of containers that
are pushed to the minikube local Docker registry. Those containers are also
automatically deployed on the Kubernetes cluster.
After the process is completed run the below command and check
the output is matching the one in the example:

```bash
kubectl get all
```

You have successfully installed the Dataset Lifecycle Framework. Easy,
isn't it?

### Create a dataset off an S3 bucket
Create a file named `dataset.yaml` with
following content. Adjust the values to match your environment.

```yaml
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: my-dataset
spec:
  local:
    type: "COS"
      accessKeyID: "your Key ID"
      secretAccessKey: "Your Secret access key"
      endpoint: "https://your.s3.endpoint.url"
      bucket: "bucket-name"
      region: "" #it can be empty
```

run

```bash
$ kubectl create -f dataset.yaml
```

to submit the dataset creation to the kubernetes cluster. From now on anyone with access to this dataset will be able to use it right away without the need for knowing details. The above step should be performed by a Cluster administrator to make sure the dataset is created in the correct namespace. 

### Label your POD and enjoy using your dataset:

Create a pod description (`pod.yaml`) labeled as in the example below:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    dataset.0.id: "my-dataset"
    dataset.0.useas: "mount"
spec:
  containers:
    - name: nginx
      image: busybox
      command:
        - sleep
        - "3600"
      volumeMounts:
        - mountPath: "/mnt/dataset1"
          name: "my-dataset"
```

make sure the dataset id in the labels matches the name used when creating the
dataset resource.

Run the pod

```bash
$ kubectl create -f pod.yaml
```

the container inside the pod will have the dataset automatically mounted at the
location specified in the yaml (`/mnt/dataset1`).

You have deployed your first pod linked to a Dataset. Remember that from now
on all the pods using that very same dataset will just be ready to go with no
needs for the user to configure access to the data source.

## Next Steps

The current release only provides support for datasets stored as S3 buckets and
requires users/administrators to input the access iformation for the dataset. We
are working on enabling the capability of fetching the dataset information from
a central catalog to facilitate access to organization-eide data lakes. Stay
tuned for updates.
