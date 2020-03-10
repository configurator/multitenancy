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

// NewConfigMapForTenant returns a new configmap object based on the provided
// multitenancy and tenant definitions.
func NewConfigMapForTenant(mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            mt.GetCombinedName(tenant),
			Namespace:       tenant.GetNamespace(),
			Labels:          util.TenantLabels(mt, tenant, nil),
			OwnerReferences: []metav1.OwnerReference{tenant.OwnerReference()},
		},
		Data: tenant.GetData(),
	}
	return configMap, util.AddCreationAnnotation(&configMap.ObjectMeta, configMap)
}

// GetConfigMapForTenant retrieves the configmap for a tenant from the api servers.
func GetConfigMapForTenant(c client.Client, tenant *confiv1.Tenant) (*corev1.ConfigMap, error) {
	cmList := &corev1.ConfigMapList{}
	if err := c.List(context.TODO(), cmList, client.InNamespace(tenant.GetNamespace()), tenant.Selector()); err != nil {
		return nil, err
	}
	if len(cmList.Items) != 1 {
		return nil, fmt.Errorf("Either none or too many matching configmaps for selector %v: %d matches", tenant.Selector(), len(cmList.Items))
	}
	return &cmList.Items[0], nil
}
