package controller

import (
	"github.com/configurator/multitenancy/pkg/controller/multitenancy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, multitenancy.Add)
}
