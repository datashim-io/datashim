[![Go Report Card](https://goreportcard.com/report/github.com/datashim-io/datashim)](https://goreportcard.com/report/github.com/datashim-io/datashim)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4821/badge)](https://bestpractices.coreinfrastructure.org/projects/4821)
# Datashim
<img src="./docs/pictures/lfaidata-project-badge-incubation-color.png" alt="drawing" width="200"/>

>Our Framework introduces the **Dataset** CRD which is a pointer to existing S3 and NFS data sources. It includes the
>necessary logic to map these Datasets into Persistent Volume Claims and ConfigMaps which users can reference in their
>pods, letting them focus on the workload development and not on configuring/mounting/tuning the data access. Thanks to
>[Container Storage Interface](https://kubernetes-csi.github.io/docs/) it is extensible to support additional data sources in the future.

![DLF](./docs/pictures/dlf.png)

A Kubernetes Framework to provide easy access to S3 and NFS **Datasets** within pods. Orchestrates the provisioning of
**Persistent Volume Claims** and **ConfigMaps** needed for each **Dataset**. Find more details in our [FAQ](https://datashim-io.github.io/datashim/FAQ/)

## Quickstart

In order to quickly deploy DLF, based on your environment execute **one** of the following commands:

- **Kubernetes/Minikube**
```bash
kubectl apply -f https://raw.githubusercontent.com/datashim-io/datashim/master/release-tools/manifests/dlf.yaml
```
- **Kubernetes on IBM Cloud**
```bash
kubectl apply -f https://raw.githubusercontent.com/datashim-io/datashim/master/release-tools/manifests/dlf-ibm-k8s.yaml
```
- **Openshift**
```bash
kubectl apply -f https://raw.githubusercontent.com/datashim-io/datashim/master/release-tools/manifests/dlf-oc.yaml
```
- **Openshift on IBM Cloud**
```bash
kubectl apply -f https://raw.githubusercontent.com/datashim-io/datashim/master/release-tools/manifests/dlf-ibm-oc.yaml
```

Wait for all the pods to be ready :)
```bash
kubectl wait --for=condition=ready pods -l app.kubernetes.io/name=dlf -n dlf
```

As an **optional** step, label the namespace(or namespaces) you want in order have the pods labelling functionality (see below).
```bash
kubectl label namespace default monitor-pods-datasets=enabled
```

_In case don't have an existing S3 Bucket follow our wiki to [deploy an Object Store](https://github.com/datashim-io/datashim/wiki/Deployment-and-Usage-of-S3-Object-Stores)
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

**Note:** We recommend using secrets to pass your S3/Object Storage Service credentials to Datashim, as shown in [this example](./examples/templates/example-dataset-s3-provision.yaml).

Feel free to explore our [other examples](./examples)

## Questions

The wiki and [Frequently Asked Questions](https://datashim-io.github.io/datashim/FAQ) documents are a bit out of date. We recommend browsing [the issues](https://github.com/datashim-io/datashim/issues?q=is%3Aissue+label%3Aquestion) for previously answered questions. Please open an issue if you are not able to find the answers to your questions, or if you have discovered a bug. 

## Contributing

We welcome all contributions to Datashim. Please read [this document](./docs/GitWorkflow.md) for setting up a Git workflow for contributing to Datashim. This project uses [DCO (Developer Certificate of Origin)](https://github.com/apps/dco) to certify code ownership and contribution rights. 

If you use VSCode, then we have [recommendations for setting it up for development](./docs/GolangVSCodeGit.md). 

If you have an idea for a feature request, please open an issue. Let us know in the issue description the problem or the pain point, and how the proposed feature would help solve it. If you are looking to contribute but you don't know where to start, we recommend looking at the open issues first. Thanks!

