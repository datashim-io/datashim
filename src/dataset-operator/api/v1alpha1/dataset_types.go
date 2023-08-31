/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const (
	StatusEmpty    = ""
	StatusInitial  = "Initializing"
	StatusPending  = "Pending"
	StatusOK       = "OK"
	StatusDisabled = "Disabled"
	StatusFail     = "Failed"
)

// DatasetSpec defines the desired state of Dataset
type DatasetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Dataset. Edit dataset_types.go to remove/update
	Local  map[string]string `json:"local,omitempty"`
	Remote map[string]string `json:"remote,omitempty"`
	// TODO temp definition for archive
	Type    string `json:"type,omitempty"`
	Url     string `json:"url,omitempty"`
	Format  string `json:"format,omitempty"`
	Extract string `json:"extract,omitempty"`
}

type DatasetStatusCondition struct {
	Status string `json:"status,omitempty"`
	Info   string `json:"info,omitempty"`
}

// DatasetStatus defines the observed state of Dataset
type DatasetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Caching   DatasetStatusCondition `json:"caching,omitempty"`
	Provision DatasetStatusCondition `json:"provision,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+genClient
//+genClient: noStatus
// Dataset is the Schema for the datasets API
type Dataset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasetSpec   `json:"spec,omitempty"`
	Status DatasetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatasetList contains a list of Dataset
type DatasetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dataset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dataset{}, &DatasetList{})
}
