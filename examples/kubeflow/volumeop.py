# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
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
    apiVersion: datashim.io/v1alpha1
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

## unfortunately can't see another way than this https://stackoverflow.com/questions/50658360/how-can-i-load-null-from-yml-as-none-class-str-not-class-nonetype
## you can just emit region in the template above instead (if its empty)
def convert_none_to_str(data):
    if isinstance(data, list):
        data[:] = [convert_none_to_str(i) for i in data]
    elif isinstance(data, dict):
        for k, v in data.items():
            data[k] = convert_none_to_str(v)
    return '' if data is None else data

if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(volume_op_dag, __file__ + ".tar.gz")
