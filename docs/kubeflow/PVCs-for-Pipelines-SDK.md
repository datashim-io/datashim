We will show how you can use DLF to provision Persistent Volume Claims via DLF so you can use it within Pipelines SDK.

## Requirements
You have kubeflow installed and you can deploy pipelines using the Pipeline SDK.

Make sure you first follow the guide for [Installation](https://github.com/IBM/dataset-lifecycle-framework/wiki/Installation)

We will just how you can adopt the examples located in [contrib/volume_ops](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/volume_ops)

**NOTE**: For this guide you can use both an empty and pre-populated with data bucket.

## Example with creation of Dataset before the pipeline execution

First you need to create a Dataset to point to the bucket you want to use. Create a file that looks like this:
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

Now within `{my-namespace}` you will find a PVC which you can use within your pipelines SDK without a problem.

You can see the [example](https://github.com/IBM/dataset-lifecycle-framework/blob/master/examples/kubeflow/volumeop.py) below which can use the PVC which was created out of your dataset.
``` python
import kfp
import kfp.dsl as dsl
from kfp.dsl import PipelineVolume


@dsl.pipeline(
    name="Volume Op DAG",
    description="The second example of the design doc."
)
def volume_op_dag():

    dataset = PipelineVolume("your-dataset")

    step1 = dsl.ContainerOp(
        name="step1",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo 1|tee /data/file1"],
        pvolumes={"/data": dataset}
    )

    step2 = dsl.ContainerOp(
        name="step2",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["cp /data/file1 /data/file2"],
        pvolumes={"/data": step1.pvolume}
    )

    step3 = dsl.ContainerOp(
        name="step3",
        image="library/bash:4.4.23",
        command=["cat", "/mnt/file1", "/mnt/file2"],
        pvolumes={"/mnt": step2.pvolume}
    )



if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(volume_op_dag, __file__ + ".tar.gz")
```

## Example with creation of Dataset as part of the pipeline execution

If instead you want to create a Dataset as part of your pipeline, you can create the Dataset yaml and invoke a `ResourceOp`.

Before that you need to make sure that the service account `pipeline-runner` in namespace `kubeflow` can create/delete Datasets, so make sure you execute `kubectl apply -f examples/kubeflow/pipeline-runner-binding.yaml` before running the pipeline. The example rolebinding definition is in [examples/kubeflow/pipeline-runner-binding.yaml](https://github.com/IBM/dataset-lifecycle-framework/blob/master/examples/kubeflow/pipeline-runner-binding.yaml)

In the following pipeline we are creating the Dataset in step0 and then proceed to step1 to use it:

``` python
import kfp.dsl as dsl
import yaml
from kfp.dsl import PipelineVolume

# Make sure that you have applied ./pipeline-runner-binding.yaml
# or any serviceAccount that should be allowed to create/delete datasets

@dsl.pipeline(
    name="Volume Op DAG",
    description="The second example of the design doc."
)
def volume_op_dag():

    datasetName = "your-dataset"
    dataset = PipelineVolume(datasetName)

    step0 = dsl.ResourceOp(name="dataset-creation",k8s_resource=get_dataset_yaml(
        datasetName,
        "XXXXXXXXXXXXXXX",
        "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
        "http://your_endpoint.com",
        "bucket-name",
        ""
    ))

    step1 = dsl.ContainerOp(
        name="step1",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo 1|tee /data/file1"],
        pvolumes={"/data": dataset}
    ).after(step0)

    step2 = dsl.ContainerOp(
        name="step2",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["cp /data/file1 /data/file2"],
        pvolumes={"/data": step1.pvolume}
    )

    step3 = dsl.ContainerOp(
        name="step3",
        image="library/bash:4.4.23",
        command=["cat", "/mnt/file1", "/mnt/file2"],
        pvolumes={"/mnt": step2.pvolume}
    )

def get_dataset_yaml(name,accessKey,secretAccessKey,endpoint,bucket,region):
    print(region)
    dataset_spec = f"""
    apiVersion: com.ie.ibm.hpsys/v1alpha1
    kind: Dataset
    metadata:
      name: {name}
    spec:
      local:
        type: "COS"
        accessKeyID: {accessKey}
        secretAccessKey: {secretAccessKey}
        endpoint: {endpoint}
        bucket: {bucket}
        region: {region}
    """
    data = yaml.safe_load(dataset_spec)
    convert_none_to_str(data)
    return data
```