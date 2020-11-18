We will show how you can use DLF to store/serve trained models using S3 Buckets.

## Requirements

You have permissions in a namespace where you can use for kubeflow (to create TFJobs, deployments etc)
Lets assume the namespace you can use is `{my-namespace}`. Feel free to change accordingly.

Make sure you first follow the guide for [Installation](https://github.com/IBM/dataset-lifecycle-framework/wiki/Installation)

We will loosely follow the example posted in [mnist_vanilla_k8s.ipynb](https://github.com/kubeflow/examples/blob/master/mnist/mnist_vanilla_k8s.ipynb)

**NOTE:** All example yaml files mentioned in the wiki are also available in [examples/kubeflow](https://github.com/IBM/dataset-lifecycle-framework/tree/fixed-caching/examples/kubeflow)

## Build model container

There is a delta between existing distributed mnist examples and what's needed to run well as a TFJob.
We will skip the kaniko part and just build and use the Dockerfile and model.py in [examples/kubeflow](https://github.com/IBM/dataset-lifecycle-framework/tree/fixed-caching/examples/kubeflow)

``` bash
cd examples/kubeflow
docker build -t {MY-REGISTRY}/mnist-model -f Dockerfile.model .
docker push {MY-REGISTRY}/mnist-model
```

In case you use an authenticated registry, follow the instructions in [configure-docker-credentials](https://github.com/kubeflow/examples/tree/master/mnist#configure-docker-credentials)

## Create an S3 Bucket and its Dataset

If you have an existing s3 bucket you can use, please proceed with this one.
Otherwise follow the instructions in [Configure IBM COS Storage](https://github.com/kubeflow/examples/blob/master/mnist/mnist_vanilla_k8s.ipynb)

Now we need to create a dataset to point to the newly created bucket. Create a file that looks like this:
``` yaml
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: your-dataset
spec:
  local:
    type: "COS"
    accessKeyID: "access_key_id"
    secretAccessKey: "secret_access_key"
    endpoint: "https://YOUR_ENDPOINT"
    bucket: "YOUR_BUCKET"
    region: "" #it can be empty
```
Now just execute:
``` bash
kubectl create -f my-dataset.yaml -n {my-namespace}
```

## Launch a TFJob

Now we are ready to launch a tfjob in a much less verbose way since DLF takes care of mounting the dataset and providing access to the tensorflow pod:
``` yaml
apiVersion: kubeflow.org/v1
kind: TFJob
metadata:
  name: my-train
spec:
  tfReplicaSpecs:
    Ps:
      replicas: 1
      template:
        metadata:
          labels:
           dataset.0.id: "your-dataset"
           dataset.0.useas: "mount"
          annotations:
            sidecar.istio.io/inject: "false"
        spec:
          serviceAccount: default-editor
          containers:
            - name: tensorflow
              command:
                - python
                - /opt/model.py
                - --tf-model-dir=/mnt/datasets/your-dataset/mnist
                - --tf-export-dir=/mnt/datasets/your-dataset/mnist/export
                - --tf-train-steps=200
                - --tf-batch-size=100
                - --tf-learning-rate=0.1
              image: yiannisgkoufas/mnist
              workingDir: /opt
              resources:
                limits:
                  ephemeral-storage: "10Gi"
                requests:
                  ephemeral-storage: "10Gi"
          restartPolicy: OnFailure
    Chief:
      replicas: 1
      template:
        metadata:
          labels:
            dataset.0.id: "your-dataset"
            dataset.0.useas: "mount"
          annotations:
            sidecar.istio.io/inject: "false"
        spec:
          serviceAccount: default-editor
          containers:
            - name: tensorflow
              resources:
                limits:
                  ephemeral-storage: "10Gi"
                requests:
                  ephemeral-storage: "10Gi"
              command:
                - python
                - /opt/model.py
                - --tf-model-dir=/mnt/datasets/your-dataset/mnist
                - --tf-export-dir=/mnt/datasets/your-dataset/mnist/export
                - --tf-train-steps=200
                - --tf-batch-size=100
                - --tf-learning-rate=0.1
              image: yiannisgkoufas/mnist
          restartPolicy: OnFailure
    Worker:
      replicas: 1
      template:
        metadata:
          labels:
            dataset.0.id: "your-dataset"
            dataset.0.useas: "mount"
          annotations:
            sidecar.istio.io/inject: "false"
        spec:
          serviceAccount: default-editor
          containers:
            - name: tensorflow
              command:
                - python
                - /opt/model.py
                - --tf-model-dir=/mnt/datasets/your-dataset/mnist
                - --tf-export-dir=/mnt/datasets/your-dataset/mnist/export
                - --tf-train-steps=200
                - --tf-batch-size=100
                - --tf-learning-rate=0.1
              image: yiannisgkoufas/mnist
              workingDir: /opt
          restartPolicy: OnFailure
```
Make sure to replace `your-dataset` with the name of your dataset. Create the TFJob like that:
``` bash
kubectl create -f tfjob.yaml -n {my-namespace}
```
You should see the job running and the model stored in the end in the remote S3 bucket.

## View the Model in Tensorboard

You can inspect the model you created and stored in the remote S3 bucket by creating the following yaml file which again leverages the Dataset created.
``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mnist-tensorboard
  name: mnist-tensorboard
spec:
  selector:
    matchLabels:
      app: mnist-tensorboard
  template:
    metadata:
      labels:
        app: mnist-tensorboard
        version: v1
        dataset.0.id: "your-dataset"
        dataset.0.useas: "mount"
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      serviceAccount: default-editor
      containers:
        - command:
            - /usr/local/bin/tensorboard
            - --logdir=/mnt/datasets/your-dataset/mnist
            - --port=80
          image: tensorflow/tensorflow:1.15.2-py3
          name: tensorboard
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: mnist-tensorboard
  name: mnist-tensorboard
spec:
  ports:
    - name: http-tb
      port: 80
      targetPort: 80
  selector:
    app: mnist-tensorboard
  type: ClusterIP
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: mnist-tensorboard
spec:
  gateways:
    - kubeflow/kubeflow-gateway
  hosts:
    - '*'
  http:
    - match:
        - uri:
            prefix: /mnist/default/tensorboard/
      rewrite:
        uri: /
      route:
        - destination:
            host: mnist-tensorboard.default.svc.cluster.local
            port:
              number: 80
      timeout: 300s
```
Create the deployment:
``` bash
kubectl create -f tensorboard.yaml -n {my-namespace}
```
You can expose the service and access it remotely as described here: [Tensorboard access](https://github.com/kubeflow/examples/blob/master/mnist/mnist_vanilla_k8s.ipynb)

## Model Serving Using [KFServing](https://www.kubeflow.org/docs/components/serving/kfserving/)

You can leverage DLF to run the inference service on the model you trained using KFServing as follows:
``` yaml
apiVersion: "serving.kubeflow.org/v1alpha2"
kind: "InferenceService"
metadata:
  name: "mnist-sample"
spec:
  default:
    predictor:
      tensorflow:
        storageUri: "pvc://your-dataset/mnist/export"
```
Create the yaml:
``` bash
kubectl create -f kfserving-inference.yaml -n {my-namespace}
```
## Model Serving Using Tensorflow Serving

Again you can leverage DLF to serve the model you trained. 
``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mnist
  name: tensorflow-serving
spec:
  selector:
    matchLabels:
      app: mnist-model
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        app: mnist-model
        version: v1
        dataset.0.id: "your-dataset"
        dataset.0.useas: "mount"
    spec:
      serviceAccount: default-editor
      containers:
        - args:
            - --port=9000
            - --rest_api_port=8500
            - --model_name=mnist
            - --model_base_path=/mnt/datasets/your-dataset/mnist/export
          command:
            - /usr/bin/tensorflow_model_server
          env:
            - name: modelBasePath
              value: /mnt/datasets/your-dataset/mnist/export
          image: tensorflow/serving:1.15.0
          imagePullPolicy: IfNotPresent
          livenessProbe:
            initialDelaySeconds: 30
            periodSeconds: 30
            tcpSocket:
              port: 9000
          name: mnist
          ports:
            - containerPort: 9000
            - containerPort: 8500
          resources:
            limits:
              cpu: "4"
              memory: 4Gi
            requests:
              cpu: "1"
              memory: 1Gi
          volumeMounts:
            - mountPath: /var/config/
              name: model-config
      volumes:
        - configMap:
            name: tensorflow-serving
          name: model-config
---
apiVersion: v1
kind: Service
metadata:
  annotations:
    prometheus.io/path: /monitoring/prometheus/metrics
    prometheus.io/port: "8500"
    prometheus.io/scrape: "true"
  labels:
    app: mnist-model
  name: tensorflow-serving
spec:
  ports:
    - name: grpc-tf-serving
      port: 9000
      targetPort: 9000
    - name: http-tf-serving
      port: 8500
      targetPort: 8500
  selector:
    app: mnist-model
  type: ClusterIP
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: tensorflow-serving
data:
  monitoring_config.txt: |-
    prometheus_config: {{
      enable: true,
      path: "/monitoring/prometheus/metrics"
    }}
```
Now create the deployment:
``` bash
kubectl create -f tensorflow-serving -n {my-namespace}
```

If you want to deploy the demo with the MNIST UI follow the instructions in [MNIST UI](https://github.com/kubeflow/examples/blob/master/mnist/mnist_vanilla_k8s.ipynb)
