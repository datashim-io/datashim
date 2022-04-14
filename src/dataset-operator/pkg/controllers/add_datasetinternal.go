package controller

import (
	"github.com/datashim-io/datashim/src/dataset-operator/pkg/controller/datasetinternal"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, datasetinternal.Add)
}
