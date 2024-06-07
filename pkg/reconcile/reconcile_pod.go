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

// ReconcilePod will ensure a pod exists for a tenant, and that its spec matches
// the desired configuration. If there has been a configmap update and the pod
// still requests a data volume, it will also be recreated.
func ReconcilePod(c client.Client, reqLogger logr.Logger, mt *confiv1.MultiTenancy, pod *corev1.Pod, recreatedConfig bool) error {
	// Check if this Pod already exists
	found := &corev1.Pod{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Pod doesn't exist - create it
			reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			return c.Create(context.TODO(), pod)
		}
		// A different error when trying to get the pod
		return err
	}

	// Pod already exists - check if its spec is identical to what we would create
	if podNeedsRecreate(reqLogger, mt, recreatedConfig, pod, found) {
		// Pod spec is different - recreate it
		// This will trigger recreation because we watch for pod deletions
		reqLogger.Info("Reconciler found pod already exists - recreating pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		return c.Delete(context.TODO(), found)
	}

	reqLogger.Info("Pod spec and config are in sync - no changes necessary", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return nil
}

// podNeedsRecreate returns true if any of the conditions for which we'd want to
// recreate a pod exist.
func podNeedsRecreate(reqLogger logr.Logger, mt *confiv1.MultiTenancy, recreatedConfig bool, pod *corev1.Pod, found *corev1.Pod) bool {
	if recreatedConfig && mt.HasResourceVolume() {
		// The config was recreated, and tenantResourceVolume is specified
		// we recreate the pod, so the mount is correct
		reqLogger.Info("Pod needs to be recreated because the configMap has changed", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
		return true
	}

	if util.AnnotationCreationSpecsEqual(pod.Annotations, found.Annotations) {
		reqLogger.Info("Skip reconcile: Pod already exists and doesn't need to change", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
		return false
	}

	reqLogger.Info("Pod needs to be recreated because its spec has changed", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return true
}
