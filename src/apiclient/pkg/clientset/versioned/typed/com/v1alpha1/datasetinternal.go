/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	scheme "github.com/datashim-io/datashim/src/apiclient/pkg/clientset/versioned/scheme"
	v1alpha1 "github.com/datashim-io/datashim/src/dataset-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// DatasetInternalsGetter has a method to return a DatasetInternalInterface.
// A group's client should implement this interface.
type DatasetInternalsGetter interface {
	DatasetInternal(namespace string) DatasetInternalInterface
}

// DatasetInternalInterface has methods to work with DatasetInternal resources.
type DatasetInternalInterface interface {
	Create(ctx context.Context, datasetInternal *v1alpha1.DatasetInternal, opts v1.CreateOptions) (*v1alpha1.DatasetInternal, error)
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.DatasetInternal, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.DatasetInternalList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	DatasetInternalExpansion
}

// datasetInternals implements DatasetInternalInterface
type datasetInternal struct {
	client rest.Interface
	ns     string
}

// newDatasetInternals returns a DatasetInternals
func newDatasetInternal(c *ComV1alpha1Client, namespace string) *datasetInternal {
	return &datasetInternal{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the datasetInternal, and returns the corresponding datasetInternal object, and an error if there is any.
func (c *datasetInternal) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.DatasetInternal, err error) {
	result = &v1alpha1.DatasetInternal{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("datasetinternal").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of DatasetInternals that match those selectors.
func (c *datasetInternal) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.DatasetInternalList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.DatasetInternalList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("datasetinternal").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested datasetInternals.
func (c *datasetInternal) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("datasetinternal").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a datasetInternal and creates it.  Returns the server's representation of the datasetInternal, and an error, if there is any.
func (c *datasetInternal) Create(ctx context.Context, datasetInternal *v1alpha1.DatasetInternal, opts v1.CreateOptions) (result *v1alpha1.DatasetInternal, err error) {
	result = &v1alpha1.DatasetInternal{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("datasetinternal").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(datasetInternal).
		Do(ctx).
		Into(result)
	return
}
