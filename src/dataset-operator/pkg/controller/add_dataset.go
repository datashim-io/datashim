package controller

import (
	"github.com/IBM/dataset-lifecycle-framework/src/dataset-operator/pkg/controller/dataset"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, dataset.Add)
}
