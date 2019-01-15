package multitenancy

import (
	"context"
	"encoding/json"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	"github.com/go-logr/logr"
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

const creationSpecAnnotationKey = "confi.gurator.com/multitenancy-creation-spec"
const createdByLabel = "confi.gurator.com/multitenancy-creation-spec"

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

	// Watch for changes to secondary resource Pods and requeue the owner MultiTenancy
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &confiv1.MultiTenancy{},
	})
	if err != nil {
		return err
	}

	// err = c.Watch(&source.Kind{Type: }, &handler.EnqueueRequestForObject{})
	// if err != nil {
	// 	return err
	// }

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

func (r *ReconcileMultiTenancy) loadResources(resourceType string) ([]string, map[string]map[string]string) {
	names := []string{
		"item-one",
		"item-two",
	}

	data := map[string]map[string]string{
		"item-one": map[string]string{
			"hello": "world",
			"name":  "item-one",
		},
		"item-two": map[string]string{
			"a-file": "contents",
		},
	}

	return names, data
}

// Reconcile reads that state of the cluster for a MultiTenancy object and makes changes based on the state read
// and what is in the MultiTenancy.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMultiTenancy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling MultiTenancy")

	// Fetch the MultiTenancy instance
	instance := &confiv1.MultiTenancy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	names, data := r.loadResources(instance.Spec.TenancyKind)

	// Define what we'll be creating
	items, err := r.createItems(reqLogger,
		instance,
		names,
		data)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.deleteExtraItems(reqLogger, instance, items)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMultiTenancy) createItems(logger logr.Logger, cr *confiv1.MultiTenancy, resourceNames []string, data map[string]map[string]string) ([]runtime.Object, error) {
	result := []runtime.Object{}
	for _, name := range resourceNames {
		objects, err := r.createItemsForResource(logger, cr, name, data[name])
		if err != nil {
			return nil, err
		}
		result = append(result, objects...)
	}
	return result, nil
}

// createItems returns all the items we would create if this were a new deployment
// these items should later be compared with the actual state to reconcile
func (r *ReconcileMultiTenancy) createItemsForResource(log logr.Logger, cr *confiv1.MultiTenancy, resourceName string, data map[string]string) ([]runtime.Object, error) {
	combinedName := cr.Name + "-" + resourceName

	// Create ConfigMap for the data volume

	configMapMetadata := metav1.ObjectMeta{
		Name:      combinedName,
		Namespace: cr.Namespace,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: configMapMetadata,
		Data:       data,
	}

	addCreatedByLabel(&configMap.ObjectMeta, combinedName)
	err := addCreationAnnotation(&configMap.ObjectMeta, configMap)
	if err != nil {
		return nil, err
	}

	recreatedConfig, err := r.reconcileConfigMap(log, cr, configMap)
	if err != nil {
		return nil, err
	}

	// Create a Pod for the workload

	metadata := *cr.Spec.Template.ObjectMeta.DeepCopy()
	metadata.Namespace = cr.Namespace
	metadata.Name = combinedName

	spec := cr.Spec.Template.Spec.DeepCopy()

	tenantResourceVolume := cr.Spec.TenantResourceVolume
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

	pod := &corev1.Pod{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	addCreatedByLabel(&pod.ObjectMeta, combinedName)
	// we add the configmap version before we add the creation annotation, so that if it changes
	// a recreation is forced
	err = addCreationAnnotation(&pod.ObjectMeta, pod)
	if err != nil {
		return nil, err
	}
	err = r.reconcilePod(log, cr, pod, recreatedConfig)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *ReconcileMultiTenancy) reconcileConfigMap(reqLogger logr.Logger, cr *confiv1.MultiTenancy, configMap *corev1.ConfigMap) (bool, error) {
	reqLogger.Info(`Reconciling ConfigMap`, `ConfigMap`, configMap.ObjectMeta)

	// Set MultiTenancy instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, configMap, r.scheme); err != nil {
		return false, err
	}

	// Check if this ConfigMap already exists
	found := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, found)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// ConfigMap doesn't exist - create it
			reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
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
		reqLogger.Info("Skip reconcile: ConfigMap already exists", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
		return false, nil
	}

	// ConfigMap spec is different - recreate it
	// This will trigger recreation because we watch for pod deletions
	reqLogger.Info("Reconciler found ConfigMap already exists - recreating ConfigMap", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
	err = r.client.Delete(context.TODO(), found)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *ReconcileMultiTenancy) reconcilePod(reqLogger logr.Logger, cr *confiv1.MultiTenancy, pod *corev1.Pod, recreatedConfig bool) error {
	reqLogger.Info(`Reconciling pod`, `Pod`, pod.ObjectMeta)

	// Set MultiTenancy instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, pod, r.scheme); err != nil {
		return err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Pod doesn't exist - create it
			reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			err = r.client.Create(context.TODO(), pod)
			return err
		}
		// A different error when trying to get the pod
		return err
	}

	// Pod already exists - check if its spec is identical to what we would create
	if !recreatedConfig {
		// If we've recrated the config we can skip the annotation check because we're always going to recreate the pod
		oldAnnotation := pod.Annotations[creationSpecAnnotationKey]
		newAnnotation := found.Annotations[creationSpecAnnotationKey]
		if oldAnnotation == newAnnotation {
			reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
			return nil
		}
	}

	// Pod spec is different - recreate it
	// This will trigger recreation because we watch for pod deletions
	reqLogger.Info("Reconciler found pod already exists - recreating pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
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

func addCreatedByLabel(metadata *metav1.ObjectMeta, value string) {
	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}
	metadata.Labels[createdByLabel] = value
}

func (r *ReconcileMultiTenancy) deleteExtraItems(logger logr.Logger, cr *confiv1.MultiTenancy, createdItems []runtime.Object) error {

	// list := metav1.List{}
	// err := r.client.List(context.TODO(),
	// 	&client.ListOptions{
	// 		Namespace:     cr.Namespace,
	// 		LabelSelector: labels.Everything(),
	// 	},
	// 	&list,
	// )
	// if err != nil {
	// 	return err
	// }

	// for _, item := range list.Items {
	// 	o, ok := item.Object.(metav1.Object)
	// 	if !ok {
	// 		return fmt.Errorf("Found kubernetes item is not a metav1.Object: %T", item)
	// 	}

	// 	// owners := o.GetOwnerReferences()
	// 	// for _, owner := range owners {
	// 	// 	// owner.
	// 	// }
	// }

	return nil
}
