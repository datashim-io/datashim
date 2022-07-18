module github.com/datashim-io/datashim/src/apiclient

require (
	github.com/datashim-io/datashim/src/dataset-operator v0.0.0-20220718115804-5d90fee24dff
	k8s.io/apimachinery v0.24.3
	k8s.io/client-go v0.24.3
)

replace (
	github.com/datashim-io/datashim/src/dataset-operator => ../dataset-operator
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
)

go 1.16
