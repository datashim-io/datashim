package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatasetSpec defines the desired state of Dataset
// +k8s:openapi-gen=true
// +groupName=com.ibm.ie.hpsys
type DatasetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// Conf map[string]string `json:"conf,omitempty"`
	Local  map[string]string `json:"local,omitempty"`
	Remote map[string]string `json:"remote,omitempty"`
	// TODO temp definition for archive
	Type    string `json:"type,omitempty"`
	Url     string `json:"url,omitempty"`
	Format  string `json:"format,omitempty"`
	Extract string `json:"extract,omitempty"`
}

const (
	StatusEmpty    = ""
	StatusInitial  = "Initializing"
	StatusPending  = "Pending"
	StatusOK       = "OK"
	StatusDisabled = "Disabled"
	StatusFail     = "Failed"
)

// DatasetStatusCondition defines sub-Status conditions
// +k8s:openapi-gen=true

type DatasetStatusCondition struct {
	Status string `json:"status,omitempty"`
	Info   string `json:"info,omitempty"`
}

// DatasetStatus defines the observed state of Dataset
// +k8s:openapi-gen=true

type DatasetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Caching   DatasetStatusCondition `json:"caching,omitempty"`
	Provision DatasetStatusCondition `json:"provision,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Dataset is the Schema for the datasets API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:noStatus
// +groupName=com.ibm.ie.hpsys
type Dataset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasetSpec   `json:"spec,omitempty"`
	Status DatasetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// DatasetList contains a list of Dataset
type DatasetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dataset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dataset{}, &DatasetList{})
}
