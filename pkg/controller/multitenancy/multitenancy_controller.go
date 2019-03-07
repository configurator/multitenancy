package multitenancy

import (
	"context"
	"encoding/json"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const creationSpecAnnotationKey = "confi.gurator.com/creation-spec"
const tenantLabel = "confi.gurator.com/tenant"
const multitenancyLabel = "confi.gurator.com/multitenancy"
const managedByLabel = "confi.gurator.com/manged-by"
const managedByLabelValue = "multitenancy_controller"
const managedByLabelSelector = managedByLabel + "=" + managedByLabelValue

var log = logf.Log.WithName("controller_multitenancy")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new MultiTenancy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMultiTenancy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("multitenancy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MultiTenancy
	err = c.Watch(&source.Kind{Type: &confiv1.MultiTenancy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Tenant
	err = c.Watch(&source.Kind{Type: &confiv1.Tenant{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Tenant
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &confiv1.Tenant{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource ConfigMap and requeue the owner Tenant
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &confiv1.Tenant{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMultiTenancy{}

// ReconcileMultiTenancy reconciles a MultiTenancy object
type ReconcileMultiTenancy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileMultiTenancy) loadResources() ([]confiv1.MultiTenancy, []confiv1.Tenant, error) {
	mts := &confiv1.MultiTenancyList{}
	err := r.client.List(context.TODO(), &client.ListOptions{}, mts)
	if err != nil {
		return nil, nil, err
	}

	tenants := &confiv1.TenantList{}
	err = r.client.List(context.TODO(), &client.ListOptions{}, tenants)
	if err != nil {
		return nil, nil, err
	}

	return mts.Items, tenants.Items, nil
}

// Reconcile reads that state of the cluster for a MultiTenancy object and makes changes based on the state read
// and what is in the MultiTenancy.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMultiTenancy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile request", "request", request)

	mts, tenants, err := r.loadResources()
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Loaded resources", "multitenancy count", len(mts), "tenant count", len(tenants))

	// Define what we'll be creating
	err = r.createItems(mts, tenants)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.deleteExtraItems(mts, tenants)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMultiTenancy) createItems(mts []confiv1.MultiTenancy, tenants []confiv1.Tenant) error {
	for _, mt := range mts {
		for _, tenant := range tenants {
			if mt.Spec.TenancyKind == tenant.TenancyKind {

				err := r.createItemsForResource(&mt, &tenant)
				if err != nil {
					return err
				}
			} else {
				log.Info("Skipping tenant due to TenancyKind mismatch", "tenant", tenant.Name, "mt", mt.Name, "tenant.TenancyKind", tenant.TenancyKind, "mt.Spec.TenancyKind", mt.Spec.TenancyKind)
			}
		}
	}
	return nil
}

// createItems returns all the items we would create if this were a new deployment
// these items should later be compared with the actual state to reconcile
func (r *ReconcileMultiTenancy) createItemsForResource(mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	resourceName := tenant.Name
	data := tenant.Data
	combinedName := mt.Name + "-" + resourceName

	// Create ConfigMap for the data volume

	configMapMetadata := metav1.ObjectMeta{
		Name:      combinedName,
		Namespace: mt.Namespace,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: configMapMetadata,
		Data:       data,
	}

	addLabels(&configMap.ObjectMeta, mt, tenant)
	err := r.addOwnerReferences(&configMap.ObjectMeta, mt, tenant)
	if err != nil {
		return err
	}
	err = addCreationAnnotation(&configMap.ObjectMeta, configMap)
	if err != nil {
		return err
	}
	recreatedConfig, err := r.reconcileConfigMap(mt, configMap)
	if err != nil {
		return err
	}

	// Create a Pod for the workload

	metadata := *mt.Spec.Template.ObjectMeta.DeepCopy()
	metadata.Namespace = mt.Namespace
	metadata.Name = combinedName

	spec := mt.Spec.Template.Spec.DeepCopy()

	tenantResourceVolume := mt.Spec.TenantResourceVolume
	if tenantResourceVolume != "" {
		// Add a volume mapping to a ConfigMap
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: tenantResourceVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: combinedName,
					},
				},
			},
		})
	}

	tenantNameVariable := mt.Spec.TenantNameVariable
	if tenantNameVariable != "" {
		// Add an environment variable mapping - to every container in the pod
		for i := range spec.Containers {
			spec.Containers[i].Env = append(spec.Containers[i].Env, corev1.EnvVar{
				Name:  tenantNameVariable,
				Value: resourceName,
			})
		}
	}

	pod := &corev1.Pod{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	addLabels(&pod.ObjectMeta, mt, tenant)
	err = r.addOwnerReferences(&pod.ObjectMeta, mt, tenant)
	if err != nil {
		return err
	}
	err = addCreationAnnotation(&pod.ObjectMeta, pod)
	if err != nil {
		return err
	}
	err = r.reconcilePod(mt, pod, recreatedConfig)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileMultiTenancy) reconcileConfigMap(mt *confiv1.MultiTenancy, configMap *corev1.ConfigMap) (bool, error) {
	// Check if this ConfigMap already exists
	found := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, found)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// ConfigMap doesn't exist - create it
			log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
			err = r.client.Create(context.TODO(), configMap)
			if err != nil {
				return false, err
			}
			return false, nil
		}
		// A different error when trying to get the pod
		return false, err
	}

	// ConfigMap already exists - check if its spec is identical to what we would create
	oldAnnotation := configMap.Annotations[creationSpecAnnotationKey]
	newAnnotation := found.Annotations[creationSpecAnnotationKey]
	if oldAnnotation == newAnnotation {
		// log.Info("Skip reconcile: ConfigMap already exists", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
		return false, nil
	}

	// ConfigMap spec is different - recreate it
	// This will trigger recreation because we watch for pod deletions
	log.Info("Reconciler found ConfigMap already exists - recreating ConfigMap", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
	err = r.client.Delete(context.TODO(), found)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *ReconcileMultiTenancy) reconcilePod(mt *confiv1.MultiTenancy, pod *corev1.Pod, recreatedConfig bool) error {
	// Check if this Pod already exists
	found := &corev1.Pod{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Pod doesn't exist - create it
			log.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			err = r.client.Create(context.TODO(), pod)
			return err
		}
		// A different error when trying to get the pod
		return err
	}

	// Pod already exists - check if its spec is identical to what we would create
	recreate := false

	if recreatedConfig && mt.Spec.TenantResourceVolume != "" {
		// The config was recreated, and tenantResourceVolume is specified
		// we recreate the pod, so the mount is correct
		log.Info("Pod needs to be recreated because the configMap has changed", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
		recreate = true
	}

	// If we've recrated the config we can skip the annotation check because we're always going to recreate the pod
	oldAnnotation := pod.Annotations[creationSpecAnnotationKey]
	newAnnotation := found.Annotations[creationSpecAnnotationKey]
	if oldAnnotation != newAnnotation {
		log.Info("Pod needs to be recreated because its spec has changed", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
		recreate = true
	}

	if !recreate {
		// log.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
		return nil
	}

	// Pod spec is different - recreate it
	// This will trigger recreation because we watch for pod deletions
	log.Info("Reconciler found pod already exists - recreating pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	err = r.client.Delete(context.TODO(), found)
	if err != nil {
		return err
	}
	return nil
}

func addCreationAnnotation(metadata *metav1.ObjectMeta, item runtime.Object) error {
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
	metadata.Annotations[creationSpecAnnotationKey] = stringRepresentation

	return nil
}

func addLabels(metadata *metav1.ObjectMeta, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) {
	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}
	metadata.Labels[multitenancyLabel] = mt.Name
	metadata.Labels[tenantLabel] = tenant.Name
	metadata.Labels[managedByLabel] = managedByLabelValue
}

func (r *ReconcileMultiTenancy) addOwnerReferences(metadata *metav1.ObjectMeta, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	// Set MultiTenancy instance as the owner and controller
	if err := controllerutil.SetControllerReference(tenant, metadata, r.scheme); err != nil {
		return err
	}

	// TODO: figure out a way to set multiple owner references. This may need support from the
	// kubernetes team, as currently, if setting multiple, deletes are not cascaded corrently
	// (we want the item to be deleted if _any_ owner is deleted; currently, the item would
	// only be deleted if _all_ owners are deleted)

	return nil
}

func (r *ReconcileMultiTenancy) deleteExtraItems(mts []confiv1.MultiTenancy, tenants []confiv1.Tenant) error {
	mtsNames := make(map[string]bool)
	for _, mt := range mts {
		mtsNames[mt.Name] = true
	}

	tenantNames := make(map[string]bool)
	for _, tenant := range tenants {
		tenantNames[tenant.Name] = true
	}

	opts := &client.ListOptions{}
	opts.SetLabelSelector(managedByLabelSelector)

	pods := &corev1.PodList{}
	err := r.client.List(context.TODO(), opts, pods)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		shouldDelete := false

		mt, mtLabelFound := (pod.Labels[multitenancyLabel])
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
		tenant, tenantLabelFound := (pod.Labels[tenantLabel])
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
			err = r.client.Delete(context.TODO(), &pod)
			if err != nil {
				return err
			}
		}
	}

	configmaps := &corev1.ConfigMapList{}
	err = r.client.List(context.TODO(), opts, configmaps)
	if err != nil {
		return err
	}
	for _, cm := range configmaps.Items {
		shouldDelete := false

		mt, mtLabelFound := (cm.Labels[multitenancyLabel])
		if !mtLabelFound {
			log.Info("Deleting configmap as it is missing a multitenancy label", "cm.Name", cm.Name)
			shouldDelete = true
		} else {
			_, mtFound := mtsNames[mt]
			if !mtFound {
				log.Info("Deleting configmap the multitenancy no longer exists", "cm.Name", cm.Name, "mt", mt)
				shouldDelete = true
			}
		}
		tenant, tenantLabelFound := (cm.Labels[tenantLabel])
		if !tenantLabelFound {
			log.Info("Deleting configmap as it is missing a tenant label", "cm.Name", cm.Name)
			shouldDelete = true
		} else {
			_, tenantFound := tenantNames[tenant]
			if !tenantFound {
				log.Info("Deleting configmap as the tenant no longer exists", "cm.Name", cm.Name, "tenant", tenant)
				shouldDelete = true
			}
		}

		if shouldDelete {
			err = r.client.Delete(context.TODO(), &cm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
