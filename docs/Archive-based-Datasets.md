# Prerequisites

* We will work with the branch [master](https://github.com/datashim-io/datashim/tree/master)
* You have kubectl utility installed and your account has admin rights to install service accounts etc
* For demo purposes you can use minikube.
* Install Datashim using one of the [quickstart environments](/README.md#quickstart)

# Installation
``` bash
git clone https://github.com/datashim-io/datashim.git
cd datashim
```

After you check out the project and the correct branch, proceed with the installation of minio.

_If you already have a cloud object store, you can skip this step._
``` bash
kubectl apply -n dlf -f examples/minio/
```
The above will install on the components in the `dlf` namespace.

A final step would be to create a secret named `minio-conf` in the `dlf` namespace which would point on the connection information for the cloud object store you would be using. In the case you have provisioned our demo minio instance, execute the below. In different case adopt the connection details to reflect on your setup.
``` bash
kubectl create secret generic minio-conf --from-literal='AWS_ACCESS_KEY_ID=minio' --from-literal='AWS_SECRET_ACCESS_KEY=minio123' --from-literal='ENDPOINT=http://minio-service:9000' -n dlf
```

You can check the status of the installation:
``` bash
watch kubectl get pods -n dlf
```
When all the components are ready the output should look like this:
``` bash
NAME                                READY   STATUS      RESTARTS   AGE
csi-attacher-nfsplugin-0            2/2     Running     0          3m1s
csi-attacher-s3-0                   1/1     Running     0          3m1s
csi-hostpath-attacher-0             1/1     Running     0          3m1s
csi-hostpath-provisioner-0          1/1     Running     0          3m1s
csi-hostpathplugin-0                3/3     Running     0          3m1s
csi-nodeplugin-nfsplugin-vs7d9      2/2     Running     0          3m1s
csi-provisioner-s3-0                1/1     Running     0          3m1s
csi-s3-mrndx                        2/2     Running     0          3m1s
dataset-operator-76798546cf-9d6wj   1/1     Running     0          3m1s
generate-keys-n7m5l                 0/1     Completed   0          3m1s
minio-7979c89d5c-khncd              0/1     Running     0          3m
```

# Usage

Now we can create a Dataset based on a remote archive as follows:
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: com.ie.ibm.hpsys/v1alpha1
kind: Dataset
metadata:
  name: example-dataset
spec:
  type: "ARCHIVE"
  url: "https://dax-cdn.cdn.appdomain.cloud/dax-noaa-weather-data-jfk-airport/1.1.4/noaa-weather-data-jfk-airport.tar.gz"
  format: "application/x-tar"
EOF
```
You should see now a PVC created with the same name:
```bash
$ kubectl get pvc
NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
example-dataset   Bound    pvc-c58852a6-a597-4eb8-a05b-23d9899226bf   9314Gi     RWX            csi-s3         15s
```
You can reference the dataset in the pod either as a usual PVC or by using the labels as follows:
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    dataset.0.id: "example-dataset"
    dataset.0.useas: "mount"
spec:
  containers:
    - name: nginx
      image: nginx
EOF
```
You can exec into the pod (after it has started) and inspect that the Dataset is available as follows:
```bash
$ kubectl exec -it nginx /bin/bash
root@nginx:/# ls /mnt/datasets/example-dataset/
noaa-weather-data-jfk-airport
root@nginx:/# ls /mnt/datasets/example-dataset/noaa-weather-data-jfk-airport/
LICENSE.txt  README.txt  clean_data.py  jfk_weather.csv  jfk_weather_cleaned.csv
```
