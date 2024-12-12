## Using Datasets with Secret references

While Datashim supports including bucket credentials in the Dataset definition,
**this is insecure and should be avoided**. We recommend storing credentials in
a Kubernetes Secret object, which can then be referenced in the Dataset
definition.

Given the following Secret definition:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-dataset-secret
stringData:
  accessKeyID: "ACCESS_KEY"
  secretAccessKey: "SECRET_KEY"
```

We can create a Dataset without hardcoded credentials as such:

```yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: my-dataset
spec:
  local:
    bucket: my-bucket
    endpoint: http://my-s3-endpoint
    secret-name: my-dataset-secret
    type: COS
```

## Provisioning buckets via the Dataset

When you define a Dataset, you can either use an existing bucket or ask Datashim
to create the one referenced in the Dataset automatically. This can be done by
including `provision: "true"` in the Dataset definition as shown below:

```yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: my-dataset
spec:
  local:
    provision: "true" # <----
    bucket: my-bucket
    endpoint: http://my-s3-endpoint
    secret-name: my-dataset-secret
    type: COS
```

## Creating read-only Datasets

There are circumnstances where we want people to be able to access the contents
of a bucket but not be able to modify them. While this can (_and should_) be
done by creating a set of credentials with only "reader" permissions on the
bucket, Datashim supports creating read-only Datasets by specifying the
`readonly: "true"` option, as such:

```yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: my-dataset
spec:
  local:
    readonly: "true" # <----
    bucket: my-bucket
    endpoint: http://my-s3-endpoint
    secret-name: my-dataset-secret
    type: COS
```

## Creating Datasets on bucket subpaths

In most cases, S3 credentials give users access to all buckets in an instance
and all their subpaths. When it comes to datasets, however, we might be
interested in limiting access to a particular "folder", or sub-path. When
creating a Dataset, we can specify the `folder` option to limit access, as shown
below:

```yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: my-dataset
spec:
  local:
    bucket: my-bucket/my-user/data # <----
    endpoint: http://my-s3-endpoint
    secret-name: my-dataset-secret
    type: COS
```

## Deleting a bucket on Dataset deletion

We might want to tie the lifecycle of a bucket to that of a Dataset by creating
it and deleting it along with the Dataset. In addition to the `provision` option
mentioned [earlier](#provisioning-buckets-via-the-dataset), Datashim allows
deleting a bucket when a Dataset is deleted with the `removeOnDelete` option.

```yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: my-dataset
spec:
  local:
    provision: "true"
    removeOnDelete: "true" # <----
    bucket: my-bucket
    endpoint: http://my-s3-endpoint
    secret-name: my-dataset-secret
    type: COS
```

## Creating Datasets from archives

!!! warning
    
    For using archive Datasets, a secret called `minio-conf` must be
    present in the namespace where Datashim is installed, typically `dlf`.

    To deploy a MinIO instance in the `dlf` namespace (and automatically create the `minio`
    secret) you can use the following oneliner:
    ```bash
      kubectl apply -n dlf -f https://github.com/datashim-io/datashim/raw/master/examples/minio/minio.yaml
    ```

    **NOTE: use this only as a reference point. For production, make sure sure 
    appropriate and secure credentials are used**

Datashim allows creating Datasets from archive files with the `ARCHIVE` dataset
type. The archive will be downloaded and uploaded to the S3 backing
store described by the `minio-conf` Secret. An additional option for extracting
the 

An example Dataset of the archive type is provided:

```yaml
apiVersion: datashim.io/v1alpha1
kind: Dataset
metadata:
  name: archive-dataset
spec:
  type: "ARCHIVE"
  url: "https://dax-cdn.cdn.appdomain.cloud/dax-noaa-weather-data-jfk-airport/1.1.4/noaa-weather-data-jfk-airport.tar.gz"
  format: "application/x-tar"
  extract: "true" # <---- OPTIONAL, to extract the content of the archive
```

## Next steps

<div class="grid cards" markdown>

-   :material-professional-hexagon:{ .lg .middle } __Even more!__

    ---

    You can read up about Datashim's more advanced features in our [Advanced Usage](cert-manager.md) section

    [:octicons-arrow-right-24: Advanced Usage](cert-manager.md)

-   :material-frequently-asked-questions:{ .lg .middle } __Any questions?__

    ---

    Find answers to frequently asked questions in our [FAQ](FAQ.md)

    [:octicons-arrow-right-24: FAQ](FAQ.md)

</div>
