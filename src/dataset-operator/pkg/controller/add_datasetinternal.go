package controller

import (
	"dataset-operator/pkg/controller/datasetinternal"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, datasetinternal.Add)
}
