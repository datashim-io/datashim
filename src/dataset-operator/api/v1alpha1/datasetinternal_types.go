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

// +k8s:openapi-gen=true
type CachingPlacementInfo struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// +k8s:openapi-gen=true
type CachingPlacement struct {
	Gateways      []CachingPlacementInfo `json:"gateways,omitempty"`
	DataLocations []CachingPlacementInfo `json:"datalocations,omitempty"`
}

// +k8s:openapi-gen=true
type DatasetInternalStatusCaching struct {
	Placements CachingPlacement `json:"placements,omitempty"`
}

// DatasetInternalStatus defines the observed state of DatasetInternal
type DatasetInternalStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Caching DatasetInternalStatusCaching `json:"caching,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=datasetsinternal

// DatasetInternal is the Schema for the datasetsinternal API
type DatasetInternal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasetSpec           `json:"spec,omitempty"`
	Status DatasetInternalStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatasetInternalList contains a list of DatasetInternal
type DatasetInternalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatasetInternal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatasetInternal{}, &DatasetInternalList{})
}
