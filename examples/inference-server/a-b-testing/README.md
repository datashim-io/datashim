# Using Datashim for A/B testing models

(_Note:_ Credit to
[YAML file from @zioproto](https://github.com/zioproto/kube-cheshire-cat/blob/1ae8be76e333482a2656431c9e6de59f2132c79c/kubernetes/tgi.yaml)
for TGI deployment in Kubernetes which provided the basis for the TGI deployment
shown in this example)

Large language models are the most interesting cloud workloads of the day. This
example demonstrates reducing the friction of running A/B tests on models with
Datashim. For this example, we use the open-source
[Text Generation Inference (TGI)](https://github.com/huggingface/text-generation-inference)
from [HuggingFace](https://huggingface.co/) as the inference service that loads
the models and makes it available for prompt inputs.

> [!NOTE]  
> In this tutorial we will assume you are in the `datashim-demo` namespace. To
> create this namespace and set it as your current context you can run:
>
> ```commandline
> kubectl create namespace datashim-demo
> kubectl config set-context --current --namespace=datashim-demo
> ```

## Prerequisites

There are no prerequisites needed to follow this tutorial, as it will provide
instructions to provision a local S3 endpoint and store models in it. If you
already have them, feel free to skip the optional instructions, but make sure to
update the values in the YAMLs, as they will all reference the setup we provide.

## (OPTIONAL) Creating a local object storage endpoint

The YAML we provide provisions a local MinIO instance using hardcoded
credentials.

> [!CAUTION] > **Do not use this for any real production workloads!**

From this folder, simply run:

```commandline
kubectl create namespace minio
kubectl apply -f minio.yaml
kubectl wait pod --for=condition=Ready -n minio --timeout=-1s minio
```

## Creating a Dataset

To access our data, we must first create a `Secret` containing the credentials
to access the bucket that holds our data, and then a `Dataset` object that links
configuration information to the access credentials.

> [!IMPORTANT] Make sure your active namespace is labelled with
> `monitor-pods-datasets=enabled` so that Datashim can mount volumes in the pods
> during the tutorial. Using `datashim-demo` as the namespace, run:
>
> ```commandline
> kubectl label namespace datashim-demo monitor-pods-datasets=enabled
> ```

Run

```commandline
kubectl apply -f minio-secret.yaml
kubectl apply -f minio-dataset.yaml
```

It will apply the following:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: model-weights-secret
stringData:
  accessKeyID: "ACCESS_KEY"
  secretAccessKey: "SECRET_KEY"
---
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: model-weights
spec:
  local:
    provision: "true"
    bucket: my-model
    endpoint: http://minio.minio.svc.cluster.local:9000
    secret-name: model-weights-secret
    type: COS
```

## (OPTIONAL) Adding models in the object storage

In this tutorial we simulate a Q/A team testing two different models:
[FLAN-T5-Small](https://huggingface.co/google/flan-t5-small) will be our
"stable" model, while the bigger and improved
[FLAN-T5-Base](https://huggingface.co/google/flan-t5-base) will be our "canary"
model. To load them in our MinIO instance we can run:

```
kubectl apply -f download-flan-t5-small-to-minio.yaml
kubectl wait --for=condition=complete job/download-flan-small --timeout=-1s
kubectl apply -f download-flan-t5-base-to-minio.yaml
kubectl wait --for=condition=complete job/download-flan-base --timeout=-1s
```

This will create two Jobs that will download the two models, and wait for their
completion. This may take several minutes.

> [!NOTE]  
> Using git to clone directly in `/mnt/datasets/model-weights/the-target-path/`
> would fail on OpenShift due to the default security policies. Errors such as
> `cp: can't preserve permissions` you might see in the pod logs can be safely
> ignored.

## Creating the TGI deployments

We can now create both the stable and the canary deployments of our models using
TGI. Run:

> [!NOTE]  
> If you want to use a GPU for your deployments, you can add a nodeSelector to
> your `spec.template.spec` section, as such:
>
> ```yaml
> nodeSelector:
>   nvidia.com/gpu.product: Tesla-T4
> ```
>
> Use the appropriate label that is set by the NVIDIA/AMD operator for
> Kubernetes/Openshift.

```
kubectl apply -f tgi-stable.yaml
kubectl apply -f tgi-canary.yaml
```

To create the TGI deployments. We can wait for TGI to be ready using the
command:

```commandline
kubectl wait pod -l run=text-generation-inference -l type=stable --for=condition=Ready --timeout=-1s
kubectl wait pod -l run=text-generation-inference -l type=canary --for=condition=Ready --timeout=-1s
```

## Creating the TGI service

We can now create a service that will perform a canary test on the two
deployments, leveraging their shared label `run: text-generation-inference`:

```commandline
kubectl apply -f service.yaml
```

## Validating the deployment

We can now forward the service exposing TGI as such:

```bash
kubectl port-forward --address localhost svc/text-generation-inference 8888:8080
```

And run an inference request against it with:

```bash
curl -s http://localhost:8888/generate -X POST -d '{"inputs":"The square root of x is the cube root of y. What is y to the power of 2, if x = 4?", "parameters":{"max_new_tokens":1000}}'  -H 'Content-Type: application/json' | jq -r .generated_text
```

We should see the following outputs alternate (due to the deployments having the
same number of replicas):

```
# Output by flan-t5-small
0
# Output by flan-t5-base
x = 4 * 2 = 8 x = 16 y = 16 to the power of 2
```

Sometimes, however, this does not happen when using `port-forward`. To test it within the cluster you can deploy the following pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: curl
  namespace: datashim-demo
spec:
  containers:
    - name: curl
      image: nicolaka/netshoot
      command:
        - "sleep"
      args:
        - "infinity"
  restartPolicy: Never
```

Using:

```commandline
kubectl apply -f curl
kubectl wait pod --for=condition=Ready curl --timeout=-1s
```

And then run:

```bash
kubectl exec curl -- curl -s http://text-generation-inference:8080/generate -X POST -d '{"inputs":"The square root of x is the cube root of y. What is y to the power of 2, if x = 4?", "parameters":{"max_new_tokens":1000}}'  -H 'Content-Type: application/json' | jq -r .generated_text
```


