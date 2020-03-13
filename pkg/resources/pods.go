package resources

import (
	"context"
	"fmt"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	util "github.com/configurator/multitenancy/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewPodForTenant returns a new pod spec for the given multitenancy and tenant
// definitions.
func NewPodForTenant(mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            mt.GetCombinedName(tenant),
			Namespace:       tenant.GetNamespace(),
			Labels:          util.TenantLabels(mt, tenant, nil),
			Annotations:     mt.GetPodAnnotations(),
			OwnerReferences: []metav1.OwnerReference{tenant.OwnerReference()},
		},
		Spec: mt.GetPodSpec(tenant),
	}
	return pod, util.AddCreationAnnotation(&pod.ObjectMeta, pod)
}

// GetPodForTenant retrieves the pod for a given tenant from the api servers.
func GetPodForTenant(c client.Client, tenant *confiv1.Tenant) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	if err := c.List(context.TODO(), podList, client.InNamespace(tenant.GetNamespace()), tenant.Selector()); err != nil {
		return nil, err
	}
	if len(podList.Items) != 1 {
		return nil, fmt.Errorf("Either none or too many matching pods for selector %v: %d matches", tenant.Selector(), len(podList.Items))
	}
	return &podList.Items[0], nil
}
