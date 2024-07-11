# Simplify Generative AI Model Development on Kubernetes with Datashim

> [!NOTE]  
> This tutorial is part of a Medium article that you can find
> [on Medium](https://medium.com/ibm-data-ai/simplify-generative-ai-model-development-on-kubernetes-with-datashim-cd2999682807)

(_Note:_ Credit to
[YAML file from @zioproto](https://github.com/zioproto/kube-cheshire-cat/blob/1ae8be76e333482a2656431c9e6de59f2132c79c/kubernetes/tgi.yaml)
for TGI deployment in Kubernetes which provided the basis for the TGI deployment
shown in this example)

## Prerequisites

Please read the Medium article we have written
[on Medium](https://medium.com/ibm-data-ai/simplify-generative-ai-model-development-on-kubernetes-with-datashim-cd2999682807)
understand the context of this tutorial.

Other than that, there are no prerequisites needed to follow this tutorial, as
it will provide instructions to provision a local S3 endpoint and store two
models in it. If you already have them, feel free to skip the optional
instructions, but make sure to update the values in the YAMLs, as they will all
reference the setup we provide.

## (OPTIONAL) Creating a local object storage endpoint

The YAML we provide provisions a local MinIO instance using hardcoded
credentials.

> [!CAUTION] 
> **Do not use this for any real production workloads!**

From this folder, simply run:

```commandline
kubectl create namespace minio
kubectl apply -f minio.yaml
kubectl wait pod --for=condition=Ready -n minio --timeout=-1s minio
```

## Creating the staging and production namespaces

Let us start by creating the staging and production namespaces:

```commandline
kubectl create namespace production
kubectl create namespace staging
```

To use Datashim's functionalities, we must also label them with
`monitor-pods-datasets=enabled` so that Datashim can mount volumes in the pods:

```commandline
kubectl label namespace production monitor-pods-datasets=enabled
kubectl label namespace staging monitor-pods-datasets=enabled
```

## Creating the Datasets

To access our data, we must first create a `Secret` containing the credentials
to access the bucket that holds our data, and then a `Dataset` object that links
configuration information to the access credentials.

Run

```commandline
kubectl apply -f s3-secret-prod.yaml
kubectl apply -f dataset-prod.yaml
kubectl apply -f s3-secret-staging.yaml
kubectl apply -f dataset-staging.yaml
```

To create secrets holding the access information to our local S3 endpoint and
the related Datasets you can see in the ["A use case: model development on
Kubernetes"](LINK TBA) section of the article.

## (OPTIONAL) Adding models in the object storage

In this tutorial we simulate a development team working with two different
models: [FLAN-T5-Small](https://huggingface.co/google/flan-t5-small) will be our
"production" model, while the bigger and improved
[FLAN-T5-Base](https://huggingface.co/google/flan-t5-base) will be our "staging"
model. To load them in our MinIO instance we can run:

```
kubectl apply -f download-flan-t5-small-to-minio-prod.yaml
kubectl wait -n production --for=condition=complete job/download-flan --timeout=-1s
kubectl apply -f download-flan-t5-base-to-minio-staging.yaml
kubectl wait -n staging --for=condition=complete job/download-flan --timeout=-1s
```

This will create two Jobs that will download the appropriate model for each
namespace, and wait for their completion. This may take several minutes.

> [!NOTE]  
> Using git to clone directly in `/mnt/datasets/model-weights/my-model/` would
> fail on OpenShift due to the default security policies. Errors such as
> `cp: can't preserve permissions` you might see in the pod logs can be safely
> ignored.

## Creating the TGI deployments

As we mention in the article, we can now use the same Deployment file to serve
the model in both namespaces. Run:

```
kubectl apply -n production -f deployment.yaml
kubectl apply -n staging -f deployment.yaml
```

To create the TGI deployments. We can wait for TGI to be ready using the
command:

```commandline
kubectl wait pod -n production -l run=text-generation-inference --for=condition=Ready --timeout=-1s
kubectl wait pod -n staging -l run=text-generation-inference --for=condition=Ready --timeout=-1s
```

## Creating the TGI service

We can now create a service in both namespaces as such:

```commandline
kubectl apply -n production -f service.yaml
kubectl apply -n staging -f service.yaml
```

## Validating the deployment

We can now forward the service exposing TGI as such:

```bash
kubectl port-forward -n production --address localhost svc/text-generation-inference 8888:8080 &
kubectl port-forward -n staging --address localhost svc/text-generation-inference 8889:8080 &
```

And run an inference request against it with:

```bash
curl -s http://localhost:8888/generate -X POST -d '{"inputs":"The square root of x is the cube root of y. What is y to the power of 2, if x = 4?", "parameters":{"max_new_tokens":1000}}'  -H 'Content-Type: application/json' | jq -r .generated_text
curl -s http://localhost:8889/generate -X POST -d '{"inputs":"The square root of x is the cube root of y. What is y to the power of 2, if x = 4?", "parameters":{"max_new_tokens":1000}}'  -H 'Content-Type: application/json' | jq -r .generated_text
```

The flan-t5-small should be very fast and reply with:

```
0
```

flan-t5-base will instead take a while to reply with:

```
x = 4 * 2 = 8 x = 16 y = 16 to the power of 2
```
