package controller

import (
	"github.com/configurator/resourceful-set/pkg/controller/resourcefulset"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, resourcefulset.Add)
}
