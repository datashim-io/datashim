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
// +k8s:openapi-gen=true
type DatasetInternalStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Caching DatasetInternalStatusCaching `json:"caching,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatasetInternal is the Schema for the datasetinternals API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=datasetinternals,scope=Namespaced
type DatasetInternal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasetSpec           `json:"spec,omitempty"`
	Status DatasetInternalStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatasetInternalList contains a list of DatasetInternal
type DatasetInternalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatasetInternal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatasetInternal{}, &DatasetInternalList{})
}
