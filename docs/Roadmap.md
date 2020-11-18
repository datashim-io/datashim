The order of the features/milestones represents loosely the order of which development will start.

# Noobaa Caching Plugin

The S3-to-S3 caching is currently only supported by the Ceph/Rook-based plugin. However, we have been facing various problems as it's setup/configuration is not fully dynamic the way Noobaa is.

In the wiki [Caching-Remote-Buckets-(User-Guide)](https://github.com/noobaa/noobaa-core/wiki/Caching-Remote-Buckets-(User-Guide)) we have few hints about how to provision the cache buckets and this logic would be reflected on the Noobaa Caching Plugin

# Object Bucket API

Our current approach is based on our modified version of [csi-s3](https://github.com/ctrox/csi-s3) which is not maintained. The Object Bucket API will reduce the code we have to maintain as the S3 operations would be supported in a more K8s native manner with the new API.

All the S3-related operations should be replaced with the [Object Bucket API](https://github.com/kubernetes/enhancements/pull/1383) once it's ready to be used.

# Vault-based access management

In our current approach, for the datasets which require credentials are stored in secrets. Secrets is the de-facto kubernetes solution for storing credentials. However there are some problems when it comes to datasets. We might want to restrict the access to the datasets between the users in the same namespace. We would be able to support scenarios where UserA and UserB are on the same namespace but UserA has datasets which only they can access.

Plan to leverage [TSI](https://github.com/IBM/trusted-service-identity)

# Spectrum Scale Caching Plugin

Assuming Spectrum Scale installed on hosts we could leverage [ibm-spectrum-scale-csi](https://github.com/IBM/ibm-spectrum-scale-csi) to provide the same functionality of S3 caching as Ceph-based and Noobaa-based.

# Dataset Eviction from cache

In our current approach, in the one implementation we have of a caching plugin, every dataset is being cached without priorities or checks (whether the cache is full etc). We need to tackle this. 

The most naive way to solve it is to not to use cache for a newly created dataset when the cache is full. A more sophisticated approach would be to monitor the usage of datasets and decide to evict based on some configurable policies.

# Sequential Transformation of Datasets

In our current approach, the only possible transformation we have is Dataset -> DatasetInternal -> PVCs. In the future we would like to be able to support any number of transformation of any type. So there would be plugins that can handle a flow like this:
Dataset(s3) -(caching)-> DatasetInternal(s3) -(expose)-> DatasetInternal(NFS) -> PVC
That would give the users the capability to cache and export their datasets in the format of their preference.

# Simple Scheduling Hints

Since we are aware of the nodes where a dataset is cached we can potentially offer this information to external schedulers or decorate the pods using `nodeAffinity` to assist the default Kubernetes scheduler to place the pods closer to the cached data.
This is expected to improve the performance of the pods using the specific datasets.