## Example Application

### On Minikube with Nooba
0. Examine the script `run_sample.sh` in the directory `examples/hive/sampleapp`  
If you are using Nooba as your Object Storage provider as per the main installation, then set the value of `NOOBAA_HOME`
```
$ unset S3_ENDPOINT
$ export NOOBAA_HOME="path/to/noobaa/installation"
```
If you are using any other S3 provider, please make sure that the values of `S3_ENDPOINT`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` are set
Then, examine `run_sample.sh` and change `DOCKER_REGISTRY_COMPONENTS` to the registry used for installing hive and execute this script
```
$ ./run_sample.sh
```
This will create a dataset called `bookds`, mount it inside a pod `samplepod`, and display the CSV file

### Environment Injection of Dataset Information

1. Edit the sample pod YAML file by changing this line

```
8     dataset.0.useas: "mount"
```

to
```
8     dataset.0.useas: "configmap"
```

2. Delete the old pod and recreate it

```
$ kubectl delete pod sampleapp
$ kubectl create -f samplepod.yaml
```

3. Check the environment variables in the pod

```
$ kubectl exec -it sampleapp env | grep -i bookds
```


