package garbagecollection

import (
	"context"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	util "github.com/configurator/multitenancy/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("garbage_collection")

// RunGC loads all multitenancy and tenant objects in the cluster and performs
// garbage collection on them.
func RunGC(c client.Client) error {
	mts, tenants, err := loadResources(c)
	if err != nil {
		return err
	}

	log.Info("Loaded resources", "multitenancy count", len(mts), "tenant count", len(tenants))

	mtsNames := make(map[string]struct{})
	for _, mt := range mts {
		mtsNames[mt.Name] = struct{}{}
	}

	tenantNames := make(map[string]struct{})
	for _, tenant := range tenants {
		tenantNames[tenant.Name] = struct{}{}
	}

	if err := deleteExtraPods(c, mtsNames, tenantNames); err != nil {
		return err
	}

	if err := deleteExtraConfigMaps(c, mtsNames, tenantNames); err != nil {
		return err
	}

	return nil
}

// loadResources loads all multitenancy and tenant objects in the cluster
func loadResources(c client.Client) ([]confiv1.MultiTenancy, []confiv1.Tenant, error) {
	mts := &confiv1.MultiTenancyList{}
	err := c.List(context.TODO(), mts)
	if err != nil {
		return nil, nil, err
	}

	tenants := &confiv1.TenantList{}
	err = c.List(context.TODO(), tenants)
	if err != nil {
		return nil, nil, err
	}

	return mts.Items, tenants.Items, nil
}

// deleteExtraConfigMaps garbage collects any dangling configmaps related to a multitenancy object
func deleteExtraConfigMaps(c client.Client, mtsNames map[string]struct{}, tenantNames map[string]struct{}) error {
	configmaps := &corev1.ConfigMapList{}
	err := c.List(context.TODO(), configmaps, util.ManagedBySelector())
	if err != nil {
		return err
	}
	for _, cm := range configmaps.Items {
		shouldDelete := false

		mt, mtLabelFound := (cm.Labels[confiv1.MultiTenancyLabel])
		if !mtLabelFound {
			log.Info("Deleting configmap as it is missing a multitenancy label", "cm.Name", cm.Name)
			shouldDelete = true
		} else {
			if _, mtFound := mtsNames[mt]; !mtFound {
				log.Info("Deleting configmap the multitenancy no longer exists", "cm.Name", cm.Name, "mt", mt)
				shouldDelete = true
			}
		}
		tenant, tenantLabelFound := (cm.Labels[confiv1.TenantLabel])
		if !tenantLabelFound {
			log.Info("Deleting configmap as it is missing a tenant label", "cm.Name", cm.Name)
			shouldDelete = true
		} else {
			if _, tenantFound := tenantNames[tenant]; !tenantFound {
				log.Info("Deleting configmap as the tenant no longer exists", "cm.Name", cm.Name, "tenant", tenant)
				shouldDelete = true
			}
		}

		if shouldDelete {
			err = c.Delete(context.TODO(), &cm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteExtraPods(c client.Client, mtsNames map[string]struct{}, tenantNames map[string]struct{}) error {
	pods := &corev1.PodList{}
	err := c.List(context.TODO(), pods, util.ManagedBySelector())
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		shouldDelete := false

		mt, mtLabelFound := (pod.Labels[confiv1.MultiTenancyLabel])
		if !mtLabelFound {
			log.Info("Deleting pod as it is missing a multitenancy label", "pod.Name", pod.Name)
			shouldDelete = true
		} else {
			_, mtFound := mtsNames[mt]
			if !mtFound {
				log.Info("Deleting pod the multitenancy no longer exists", "pod.Name", pod.Name, "mt", mt)
				shouldDelete = true
			}
		}
		tenant, tenantLabelFound := (pod.Labels[confiv1.TenantLabel])
		if !tenantLabelFound {
			log.Info("Deleting pod as it is missing a tenant label", "pod.Name", pod.Name)
			shouldDelete = true
		} else {
			_, tenantFound := tenantNames[tenant]
			if !tenantFound {
				log.Info("Deleting pod as the tenant no longer exists", "pod.Name", pod.Name, "tenant", tenant)
				shouldDelete = true
			}
		}

		if shouldDelete {
			err = c.Delete(context.TODO(), &pod)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
