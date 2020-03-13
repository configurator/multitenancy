package runner

import (
	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	"k8s.io/client-go/kubernetes"
)

// HookRunner represents an interface for dealing with event hooks.
// Event hook "providers" can implement this interface to provide functionality
// for the hooks.
type HookRunner interface {
	RunCreateHook(*kubernetes.Clientset, *confiv1.MultiTenancy, *confiv1.Tenant) error
	RunUpdateHook(*kubernetes.Clientset, *confiv1.MultiTenancy, *confiv1.Tenant) error
	RunDeleteHook(*kubernetes.Clientset, *confiv1.MultiTenancy, *confiv1.Tenant, []string) error
}
