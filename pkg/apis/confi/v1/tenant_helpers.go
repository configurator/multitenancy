package v1

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetTenancy returns the multitenancy object for this tenant
func (t *Tenant) GetTenancy(c client.Client) (*MultiTenancy, error) {
	mts := &MultiTenancyList{}
	if err := c.List(context.TODO(), mts, client.InNamespace(t.Namespace)); err != nil {
		return nil, err
	}
	for _, mt := range mts.Items {
		if mt.Spec.TenancyKind == t.TenancyKind {
			return &mt, nil
		}
	}
	return nil, fmt.Errorf("Could not locate tenancy kind: %s", t.TenancyKind)
}

// GetData returns the data for this tenant instance
func (t *Tenant) GetData() map[string]string {
	return t.Data
}

// OwnerReference returns an owner reference for this tenant object
func (t *Tenant) OwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         t.APIVersion,
		Kind:               t.Kind,
		Name:               t.GetName(),
		UID:                t.GetUID(),
		Controller:         boolPointer(true),
		BlockOwnerDeletion: boolPointer(true),
	}
}

// NamespacedName returns the namespaced name for this tenant object
func (t *Tenant) NamespacedName() types.NamespacedName {
	return types.NamespacedName{Name: t.GetName(), Namespace: t.GetNamespace()}
}

// Selector returns the label selectors for this tenant object's subresources.
func (t *Tenant) Selector() client.MatchingLabels {
	return client.MatchingLabels{TenantLabel: t.GetName()}
}
