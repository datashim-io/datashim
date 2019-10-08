# Dataset Operator

## Overview

This operator adds the Dataset CRD (Custom Resource Definition)
to a k8s cluster and allows you to create objects like the example
in [dataset_cr](deploy/crds/com_v1alpha1_dataset_cr.yaml)

```
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: example-dataset
spec:
  # Add fields here
  type: COS
  conf:
    onefield: valueaa
    anotherfield: bb
```

## Prerequisites

In order to build the operator there are some tools required:

- **docker** (tested version 19.03.1)
- **go** (tested version 12.7) verify by invoking `go env`
  - Instructions: https://tecadmin.net/install-go-on-centos/ 
  (applies for all linux-based OS)
- **operator-sdk** (tested version 0.9.0) verify by invoking
`operator-sdk version`
  - Instructions: https://github.com/operator-framework/operator-sdk/blob/master/doc/user/install-operator-sdk.md
- [optional] **minikube** (tested version 1.3.0) verify by
invoking `minikube version` In case you want to test the operator
locally. 

## Getting Started With the Operator

After you have configured `kubectl` to point to the correct k8s context, 
invoke `./build-minikube.sh && ./install.sh`

If you check with `kubectl get pods` you should see the operator running:
```
NAME                               READY   STATUS    RESTARTS   AGE
dataset-operator-644f8d854-dct95   1/1     Running   0          15s
```

Now you can do `kubectl create -f dataset.yaml` with a yaml which looks like this:
```
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: example-dataset
spec:
  # Add fields here
  type: COS
  conf:
    onefield: valueaa
    anotherfield: bb
```

If you check the pods now you should find:
```
NAME                               READY   STATUS    RESTARTS   AGE
example-dataset-pod                1/1     Running   0          20s
dataset-operator-644f8d854-dct95   1/1     Running   0          3m50s
```
The demo behavior of the operator is just to create a pod for every
dataset object.

Delete the new dataset object and the operator:
```
./uninstall.sh
```

## Notes

The logic for the controller can be found in 
[dataset_controller.go](pkg/controller/dataset/dataset_controller.go)
