package apis

import (
	"github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/apis/com/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
