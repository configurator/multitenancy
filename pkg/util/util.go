package util

import (
	"encoding/json"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AddCreationAnnotation adds a string representation of the given item's spec
// to the annotations on the metadata provided.
func AddCreationAnnotation(metadata *metav1.ObjectMeta, item runtime.Object) error {
	// Get the string representation of the current object
	bytes, err := json.Marshal(item)
	if err != nil {
		return err
	}
	stringRepresentation := string(bytes)

	// Add the string representation annotation to the object's metadata
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	metadata.Annotations[confiv1.CreationSpecAnnotationKey] = stringRepresentation

	return nil
}

// TenantLabels returns the resource labels for a tenant, or its dependants.
// The labels can optionally be merged with an existing map.
func TenantLabels(mt *confiv1.MultiTenancy, tenant *confiv1.Tenant, mergeWith map[string]string) map[string]string {
	var labels map[string]string
	if mergeWith != nil {
		labels = mergeWith
	} else {
		labels = make(map[string]string)
	}
	labels[confiv1.MultiTenancyLabel] = mt.GetName()
	labels[confiv1.TenantLabel] = tenant.GetName()
	labels[confiv1.ManagedByLabel] = confiv1.ManagedByLabelValue
	return labels
}

// ManagedBySelector returns a label selector for resources managed by the
// multitenancy controller.
func ManagedBySelector() client.MatchingLabels {
	return client.MatchingLabels{confiv1.ManagedByLabel: confiv1.ManagedByLabelValue}
}

// AnnotationCreationSpecsEqual returns true if the creation specs in the two
// annotation maps provided are equal.
func AnnotationCreationSpecsEqual(a1, a2 map[string]string) bool {
	return a1[confiv1.CreationSpecAnnotationKey] == a2[confiv1.CreationSpecAnnotationKey]
}

// PodIsReady returns true if the pod object is in a Ready state.
func PodIsReady(pod *v1.Pod) bool {
	if pod.Status.Phase != v1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// StringSliceContains returns true if the given string exists in the given
// string slice.
func StringSliceContains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// StringSliceRemove returns a string slice with the given element removed from
// the slice provided.
func StringSliceRemove(ss []string, s string) []string {
	for i, x := range ss {
		if x == s {
			return append(ss[:i], ss[i+1:]...)
		}
	}
	return ss
}
