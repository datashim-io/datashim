import json

import pykorm
import pytest
from kubetest import client
import os

from tests.dataset import Dataset

test_namespace = os.getenv("DLF_TESTING_NAMESPACE")

@pytest.mark.namespace(create=False, name=test_namespace)
def test_one(kube: client.TestClient):
    #make sure that the minio deployments are up and running
    minio_deployment_noregion = kube.get_deployments(labels={"app":"minio-noregion"})["minio-noregion"]
    minio_deployment_noregion.wait_until_ready(3*60)
    #once they are ready, create a dataset that points to the bucket in Minio
    pk = pykorm.Pykorm()
    dataset = Dataset(namespace=test_namespace,
                      name="mydataset",
                      local={
                          "type": "COS",
                          "accessKeyID": "minio",
                          "secretAccessKey": "minio123",
                          "endpoint": "http://minio-service-noregion."+test_namespace+":9000",
                          "bucket": "bucket",
                          "readonly": "false"
                      })
    pk.save(dataset)

    base_scripts = kube.load_configmap("./tests/base-scripts.yaml")
    kube.create(base_scripts)
    #create a simple deployment that mounts that volume and make sure the files are there
    deployment = kube.load_deployment("./tests/alpine-dataset.yaml")
    kube.create(deployment)
    deployment.wait_until_ready(timeout=3*60)
    #wait also to execute the command
    #check on the main container if the files are there and we can read them
    main_container = deployment.get_pods()[0].get_containers()[0]
    containers_logs_arr = (main_container.get_logs().splitlines())
    ##TODO report the correct values, check if false
    for log_entry in containers_logs_arr:
        try:
            json_object = json.loads(log_entry)
            if(json_object["test"]=="file_content"):
                if("this is the content of file" not in json_object["value"]):
                    pytest.fail("Unexpected contents of file")
            elif(json_object["value"] != "true"):
                pytest.fail("Failed test,",json_object["test"])
        except:
            pytest.fail(log_entry)
    pass