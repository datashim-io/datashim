## **What is the framework offering exactly?**

**One new Custom Resource Definition: the Dataset.** Essentially this CRD is a declarative way to reference an existing data source. Moreover, we provide a mount-point in user's pod for each Dataset and expose an interface for caching mechanisms to leverage.
Current implementation supports S3- and NFS-based data sources.

## **That's it? You just add one more CRD?**

**Not quite.** For every Dataset we create one Persistent Volume Claim
which users can mount directly to their pods. We have implemented
that logic as a regular Kubernetes Operator.

## **What is the motivation for this work? What problem does it solve?**

Since the introduction of Container Storage Interface, there are more
and more storage providers becoming available on Kubernetes environments.
However we feel that for the non-experienced Kubernetes users it might be
**a high barrier for them to install/maintain/configure in order to leverage**
**the available CSI plugins** and gain access to the remote data sources on their pods.

By introducing **a higher level of abstraction (Dataset) and by taking care of all the necessary work**
around invoking the appropriate CSI plugin, configuring and provisioning
the PVC we aim to improve the **User Experience of data access in Kubernetes**

## **So...you want to replace CSI?**

**On the contrary!** Every type of data source we support actually comes with its own
completely standalone CSI implementation.

We are aspiring to be **a meta-framework for the CSI plugins**.
If we have to make a comparison, we want make accessible different types of data sources 
the same way Kubeflow makes Machine Learning frameworks accessible on Kubernetes

## **Are you competing with the COSI proposal?**

**Absolutely no**. COSI aims to manage the full lifecycle of a bucket like provisioning, configuring access etc. which is beyond our scope. We just want to offer a mountpoint for COS buckets

## **Any other potential benefits you see with the framework?**

We believe that by introducing Dataset as a CRD you can accomplish higher level
orchestration and bring contributions on:
- **Performance**: We have attempted to create a pluggable caching interface like the example implementation: [Ceph Caching Plugin](https://github.com/IBM/dataset-lifecycle-framework/wiki/Ceph-Caching)
- **Security**: Another effort we are exploring is to have a common access management layer
for credentials of the different types of datasources 

## **Is anyone actually interested in the framework?**
* **European Bioinformatics Institute** ( https://www.ebi.ac.uk/ ) are running a POC with Datashim and Kubeflow on their cloud infrastructure
  * [David Yu Yuan](https://github.com/davidyuyuan) actually reached out to us after a CNCF presentation
* People from **Open Data Hub** ( https://opendatahub.io/ ) are interested in integrating Datashim in ODH
  * See relevant issue ( https://github.com/IBM/dataset-lifecycle-framework/issues/40 )
* **Pachyderm's proposal** is actually very close to the Dataset spec we are supporting.
  * Datashim is forked in their repo and is under evaluation in their repo https://github.com/pachyderm/kfdata
