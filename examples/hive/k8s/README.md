# Installing Hive in Kubernetes

In this series of steps, we will be installing Hive in Kubernetes to have a
metadata catalog that can be queried by the framework to create datasets. Hive
should be installed in the same namespace as the rest of DLF. This
guide assumes Minikube as the target Kubernetes cluster, but it is applicable to any Kubernetes/Openshift infrastructure.

## Initial steps

First, some configuration. The ObjectStorage integration of Hive requires that the endpoint be provided at the point of initial configuration. 
if you are using the Nooba install as described in the main installation guide, then all you have to do is to export the directory where Nooba is placed.

```
$ unset S3_ENDPOINT
$ export NOOBAA_HOME=path/to/Noobaa/directory
```
If you are using a different Object Storage service, then you need to set these environment variables

```
$ export S3_ENDPOINT=http://<Object_Storage_Service_Provider_URL>
$ export AWS_ACCESS_KEY_ID = "Access key for Object Storage"
$ export AWS_SECRET_ACCESS_KEY = "Secret access key for Object Storage"
```
Then, examine `Makefile` in `examples/hive/k8s` and add values for `DATASET_NAMESPACE_OPERATOR`, `DOCKER_REGISTRY_COMPONENTS`, and `DOCKER_REGISTRY_SECRET`. Please ensure that these variable values are the same as that used for installing DLF.

Now go ahead and complete the install
```
$ make minikube-install
```

Test your installation with `test-hive.sh`. Examine the script in a editor and change the values of namespace and repository variables
```
$ ./test-hive.sh
```
If the output is
```
HTTP/1.1 200 OK
```
then you can try the URL provided in a browser and verify that the Hive landing page is displayed correctly
