package resourcefulset

import (
	"context"
	"encoding/json"

	confiv1 "github.com/configurator/resourceful-set/pkg/apis/confi/v1"
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

const creationSpecAnnotationKey = "confi.gurator.com/resourcefulset-creation-spec"
const createdByLabel = "confi.gurator.com/resourceful-set-creation-spec"

var log = logf.Log.WithName("controller_resourcefulset")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ResourcefulSet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileResourcefulSet{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("resourcefulset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ResourcefulSet
	err = c.Watch(&source.Kind{Type: &confiv1.ResourcefulSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ResourcefulSet
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &confiv1.ResourcefulSet{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileResourcefulSet{}

// ReconcileResourcefulSet reconciles a ResourcefulSet object
type ReconcileResourcefulSet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ResourcefulSet object and makes changes based on the state read
// and what is in the ResourcefulSet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileResourcefulSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ResourcefulSet")

	// Fetch the ResourcefulSet instance
	instance := &confiv1.ResourcefulSet{}
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

	// Define what we'll be creating
	r.createItems(reqLogger,
		instance,
		[]string{
			"example-item",
			"another-one",
		},
		map[string]map[string]string{
			"example-item": map[string]string{
				"hello": "world",
			},
			"another-one": map[string]string{
				"a-file": "contents",
			},
		})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Add an annotation to each item, to compare versions to see if items need to be recreated

	// log.Info(`Desired items`, `items`, items)

	// list := metav1.List{}
	// err = r.client.List(context.TODO(),
	// 	&client.ListOptions{
	// 		Namespace:     instance.Namespace,
	// 		LabelSelector: labels.Everything(),
	// 	},
	// 	&list,
	// )

	// if err != nil {
	// 	return reconcile.Result{}, err
	// }

	return reconcile.Result{}, nil

	// pod, err := createPod(instance, "example-item")
	// if err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // Set ResourcefulSet instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // Check if this Pod already exists
	// found := &corev1.Pod{}
	// err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	// if err != nil && kerrors.IsNotFound(err) {
	// 	reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	// 	err = r.client.Create(context.TODO(), pod)
	// 	if err != nil {
	// 		return reconcile.Result{}, err
	// 	}

	// 	// Pod created successfully - don't requeue
	// 	return reconcile.Result{}, nil
	// } else if err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // Pod already exists - check if its spec is identical to what we would create
	// oldAnnotation := pod.Annotations[creationSpecAnnotationKey]
	// newAnnotation := found.Annotations[creationSpecAnnotationKey]
	// if oldAnnotation == newAnnotation {
	// 	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	// 	return reconcile.Result{}, nil
	// }

	// // Pod spec is different - recreate it
	// // This will trigger recreation because we watch for pod deletions
	// reqLogger.Info("Reconciler found pod already exists - recreating pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	// err = r.client.Delete(context.TODO(), found)
	// // The pod will be recreated because we're watching deletions
	// return reconcile.Result{}, nil
}

func (r *ReconcileResourcefulSet) createItems(logger logr.Logger, cr *confiv1.ResourcefulSet, resourceNames []string, data map[string]map[string]string) {
	for _, name := range resourceNames {
		r.createItemsForResource(logger, cr, name, data[name])
	}
}

// createItems returns all the items we would create if this were a new deployment
// these items should later be compared with the actual state to reconcile
func (r *ReconcileResourcefulSet) createItemsForResource(log logr.Logger, cr *confiv1.ResourcefulSet, resourceName string, data map[string]string) error {
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
		return err
	}

	recreatedConfig, err := r.reconcileConfigMap(log, cr, configMap)
	if err != nil {
		return err
	}

	// Create a Pod for the workload

	metadata := *cr.Spec.Template.ObjectMeta.DeepCopy()
	metadata.Namespace = cr.Namespace
	metadata.Name = combinedName

	spec := cr.Spec.Template.Spec.DeepCopy()

	replicationResourceVolume := cr.Spec.ReplicationResourceVolume
	if replicationResourceVolume != "" {
		// Add a volume mapping to a ConfigMap
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: replicationResourceVolume,
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
		return err
	}
	err = r.reconcilePod(log, cr, pod, recreatedConfig)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileResourcefulSet) reconcileConfigMap(reqLogger logr.Logger, cr *confiv1.ResourcefulSet, configMap *corev1.ConfigMap) (bool, error) {
	reqLogger.Info(`Reconciling ConfigMap`, `ConfigMap`, configMap.ObjectMeta)

	// Set ResourcefulSet instance as the owner and controller
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

func (r *ReconcileResourcefulSet) reconcilePod(reqLogger logr.Logger, cr *confiv1.ResourcefulSet, pod *corev1.Pod, recreatedConfig bool) error {
	reqLogger.Info(`Reconciling pod`, `Pod`, pod.ObjectMeta)

	// Set ResourcefulSet instance as the owner and controller
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
