# Serving Models stored in S3 buckets using Datashim Datasets

(_Note:_ Credit to
[YAML file from @zioproto](https://github.com/zioproto/kube-cheshire-cat/blob/1ae8be76e333482a2656431c9e6de59f2132c79c/kubernetes/tgi.yaml)
for TGI deployment in Kubernetes which provided the basis for the TGI deployment
shown in this example)

Large language models are the most interesting cloud workloads of the day. This
example demonstrates reducing the friction of loading models from an S3 bucket
using Datashim. For this example, we use the open-source
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
instructions to provision a local S3 endpoint and store a model in it. If you
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

To apply the following:

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

## (OPTIONAL) Adding a model in the object storage

In this tutorial we will use the
[FLAN-T5-Base model](https://huggingface.co/google/flan-t5-base) as our set of
weights to be loaded. To load them in our MinIO instance we can run:

```
kubectl apply -f download-flan-t5-base-to-minio.yaml
kubectl wait --for=condition=complete job/download-flan --timeout=-1s
```

To create a download `Job` and wait for its completion:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: download-flan
spec:
  template:
    metadata:
      labels:
        dataset.0.id: "model-weights"
        dataset.0.useas: "mount"
    spec:
      containers:
        - image: alpine/git
          command: ["git"]
          args:
            - "clone"
            - "https://huggingface.co/google/flan-t5-base/"
            - "/mnt/datasets/model-weights/flan-t5-base/"
          imagePullPolicy: IfNotPresent
          name: git
      restartPolicy: Never
```

## Creating the TGI pod

As anticipated, we will use TGI to serve the model. Run

```
kubectl apply tgi+service.yaml
```

To create the following `Pod` and `Service`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: text-generation-inference
  labels:
    run: text-generation-inference
    dataset.0.id: "model-weights"
    dataset.0.useas: "mount"
spec:
  containers:
    - name: text-generation-inference
      image: ghcr.io/huggingface/text-generation-inference:1.3.4
      env:
        - name: RUST_BACKTRACE
          value: "1"
      command:
        - "text-generation-launcher"
        - "--model-id"
        - "/mnt/datasets/model-weights/flan-t5-base/"
        - "--sharded"
        - "false"
        - "--port"
        - "8080"
        - "--huggingface-hub-cache"
        - "/tmp"
      ports:
        - containerPort: 8080
          name: http
      readinessProbe:
        tcpSocket:
          port: 8080
        initialDelaySeconds: 5
        periodSeconds: 5
  restartPolicy: Never
---
apiVersion: v1
kind: Service
metadata:
  name: text-generation-inference
spec:
  ports:
    - port: 8080
      protocol: TCP
      targetPort: 8080
  selector:
    run: text-generation-inference
  type: ClusterIP
```

The key lines are the labels starting with `dataset.0.` which define the
`model_weights` dataset as an input to the TGI pod and the command arguments
`"--weights-cache-override"` which indicates to TGI to load the model weights
from a specific directory. In this example, the directory location points to the
volume where the bucket will eventually be mounted
(`/mnt/datasets/model_weights`) and where the model weights will be found.

We can wait for TGI to be ready using the command:

```commandline
kubectl wait pod --for=condition=Ready text-generation-inference --timeout=-1s
```

We can also monitor the pods by looking at the logs:

```commandline
kubectl logs -f text-generation-inference
```

If all goes well, you will see the following:

```txt
...
2024-03-04T16:37:56.587319Z INFO shard-manager: text_generation_launcher: Waiting for shard to be ready... rank=0
2024-03-04T16:38:06.594424Z INFO shard-manager: text_generation_launcher: Waiting for shard to be ready... rank=0
2024-03-04T16:38:09.479212Z INFO text_generation_launcher: Server started at unix:///tmp/text-generation-server-0
2024-03-04T16:38:09.496633Z INFO shard-manager: text_generation_launcher: Shard ready in 22.918174777s rank=0
2024-03-04T16:38:09.593469Z INFO text_generation_launcher: Starting Webserver
2024-03-04T16:38:09.659675Z WARN text_generation_router: router/src/main.rs:194: no pipeline tag found for model /mnt/datasets/model-weights/flan-t5-base/
2024-03-04T16:38:09.663104Z INFO text_generation_router: router/src/main.rs:213: Warming up model
2024-03-04T16:38:11.273900Z WARN text_generation_router: router/src/main.rs:224: Model does not support automatic max batch total tokens
2024-03-04T16:38:11.273926Z INFO text_generation_router: router/src/main.rs:246: Setting max batch total tokens to 16000
2024-03-04T16:38:11.273934Z INFO text_generation_router: router/src/main.rs:247: Connected
2024-03-04T16:38:11.273942Z WARN text_generation_router: router/src/main.rs:252: Invalid hostname, defaulting to 0.0.0.0
```

This indicates that the service has been set up successfully and is ready to
reply to prompts.

## Validating the deployment

We can now forward the service exposing TGI as such:

```bash
kubectl port-forward --address localhost pod/text-generation-inference 8888:8080
```

And run an inference request against it with:

```bash
 curl -s http://localhost:8888/generate -X POST -d '{"inputs":"The square root of x is the cube root of y. What is y to the power of 2, if x = 4?", "parameters":{"max_new_tokens":1000}}'  -H 'Content-Type: application/json' | jq -r .generated_text
```

We should see the following output:

```
x = 4 * 2 = 8 x = 16 y = 16 to the power of 2
```
