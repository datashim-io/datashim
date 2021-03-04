# Datashim

[![Go Report Card](https://goreportcard.com/badge/github.com/IBM/dataset-lifecycle-framework)](https://goreportcard.com/report/github.com/IBM/dataset-lifecycle-framework)

>Our Framework introduces the **Dataset** CRD which is a pointer to existing S3 and NFS data sources. It includes the
>necessary logic to map these Datasets into Persistent Volume Claims and ConfigMaps which users can reference in their
>pods, letting them focus on the workload development and not on configuring/mounting/tuning the data access. Thanks to
>[Container Storage Interface](https://kubernetes-csi.github.io/docs/) it is extensible to support additional data sources in the future.

![DLF](./docs/pictures/dlf.png)

A Kubernetes Framework to provide easy access to S3 and NFS **Datasets** within pods. Orchestrates the provisioning of
**Persistent Volume Claims** and **ConfigMaps** needed for each **Dataset**. Find more details in our [FAQ](https://ibm.github.io/dataset-lifecycle-framework/FAQ/)

## Quickstart

In order to quickly deploy DLF, based on your environment execute **one** of the following commands:

- **Kubernetes/Minikube**
```bash
kubectl apply -f https://raw.githubusercontent.com/IBM/dataset-lifecycle-framework/master/release-tools/manifests/dlf.yaml
```
- **Kubernetes on IBM Cloud**
```bash
kubectl apply -f https://raw.githubusercontent.com/IBM/dataset-lifecycle-framework/master/release-tools/manifests/dlf-ibm-k8s.yaml
```
- **Openshift**
```bash
kubectl apply -f https://raw.githubusercontent.com/IBM/dataset-lifecycle-framework/master/release-tools/manifests/dlf-oc.yaml
```
- **Openshift on IBM Cloud**
```bash
kubectl apply -f https://raw.githubusercontent.com/IBM/dataset-lifecycle-framework/master/release-tools/manifests/dlf-ibm-oc.yaml
```

Wait for all the pods to be ready :)
```bash
kubectl wait --for=condition=ready pods -l app.kubernetes.io/name=dlf -n dlf
```

As an **optional** step, label the namespace(or namespaces) you want in order have the pods labelling functionality (see below).
```bash
kubectl label namespace default monitor-pods-datasets=enabled
```

_In case don't have an existing S3 Bucket follow our wiki to [deploy an Object Store](https://github.com/IBM/dataset-lifecycle-framework/wiki/Deployment-and-Usage-of-S3-Object-Stores)
and populate it with data._

We will create now a Dataset named `example-dataset` pointing to your S3 bucket.
```yaml
cat <<EOF | kubectl apply -f -
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
    readonly: "true" #OPTIONAL, default is false  
    region: "" #OPTIONAL
EOF
```

If everything worked okay, you should see a PVC and a ConfigMap named `example-dataset` which you can mount in your pods.
As an easier way to use the Dataset in your pod, you can instead label the pod as follows:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    dataset.0.id: "example-dataset"
    dataset.0.useas: "mount"
spec:
  containers:
    - name: nginx
      image: nginx
```

As a convention the Dataset will be mounted in `/mnt/datasets/example-dataset`. If instead you wish to pass the connection
details as environment variables, change the `useas` line to `dataset.0.useas: "configmap"`

Feel free to explore our [examples](./examples)

## FAQ

Have a look on our wiki for [Frequently Asked Questions](https://github.com/IBM/dataset-lifecycle-framework/wiki/FAQ)

## Roadmap

Have a look on our wiki for [Roadmap](https://github.com/IBM/dataset-lifecycle-framework/wiki/Roadmap)

## Contact

Reach out to us via email:
- Yiannis Gkoufas, <yiannisg@ie.ibm.com>
- Christian Pinto, <christian.pinto@ibm.com>
- Srikumar Venugopal, <srikumarv@ie.ibm.com>
- Panagiotis Koutsovasilis, <koutsovasilis.panagiotis1@ibm.com>

## Acknowledgements
This project has received funding from the European Union’s Horizon 2020 research and innovation programme under grant agreement No 825061.

[H2020 evolve](https://www.evolve-h2020.eu/).

<img src="./docs/pictures/evolve-logo.png" alt="H2020 evolve logo" width="150" height="24.07">

