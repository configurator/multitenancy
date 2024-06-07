package reconcile

import (
	"context"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	util "github.com/configurator/multitenancy/pkg/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileConfigMap will ensure a configmap for a tenant is present and matches
// the data provided in our specs.
func ReconcileConfigMap(c client.Client, reqLogger logr.Logger, mt *confiv1.MultiTenancy, configMap *corev1.ConfigMap) (bool, error) {
	// Check if this ConfigMap already exists
	found := &corev1.ConfigMap{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, found)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// ConfigMap doesn't exist - create it
			reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
			return false, c.Create(context.TODO(), configMap)
		}
		// A different error when trying to get the configmap
		return false, err
	}

	// ConfigMap already exists - check if its spec is identical to what we would create
	if util.AnnotationCreationSpecsEqual(configMap.Annotations, found.Annotations) {
		reqLogger.Info("Skip reconcile: ConfigMap already exists and doesn't need to change", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
		return false, nil
	}

	// ConfigMap spec is different - recreate it
	// This will trigger recreation because we watch for pod deletions
	reqLogger.Info("Reconciler found ConfigMap already exists - recreating ConfigMap", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
	return true, c.Delete(context.TODO(), found)
}
