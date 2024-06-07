package tenant

import (
	"context"
	"reflect"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	eventhooks "github.com/configurator/multitenancy/pkg/eventhooks"
	confireconcile "github.com/configurator/multitenancy/pkg/reconcile"
	resources "github.com/configurator/multitenancy/pkg/resources"
	util "github.com/configurator/multitenancy/pkg/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_tenant")

// Add creates a new Tenant Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTenant{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("tenant-controller", mgr, controller.Options{Reconciler: r})
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

	// Watch for changes to MultiTenancy objects, and requeue the tenants that are
	// dependent on it.
	err = c.Watch(
		&source.Kind{Type: &confiv1.MultiTenancy{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				// The object will always be a multitenancy object, but this provides an extra safety belt
				// from panics.
				mt, ok := a.Object.(*confiv1.MultiTenancy)
				if !ok {
					log.Info("Failed to assert runtime.Object to confiv1.MultiTenancy")
					return nil
				}
				log.Info("Searching for tenants affected by multitenancy update", "MultiTenancy.Name", mt.GetName(), "MultiTenancy.Namespace", mt.GetNamespace())
				reqs, err := getDependentTenants(mgr.GetClient(), mt)
				if err != nil {
					log.Error(err, "Error requeuing devices")
				}
				return reqs
			}),
		})
	if err != nil {
		return err
	}

	return nil
}

// getDependantTenants returns the list of tenants that derive their specs from
// the given multitenancy object
func getDependentTenants(c client.Client, mt *confiv1.MultiTenancy) ([]reconcile.Request, error) {
	tenants := &confiv1.TenantList{}
	if err := c.List(context.TODO(), tenants, mt.NamespaceSelector(), mt.Selector()); err != nil {
		return nil, err
	}
	reqs := make([]reconcile.Request, 0)
	for _, tenant := range tenants.Items {
		reqs = append(reqs, reconcile.Request{
			NamespacedName: tenant.NamespacedName(),
		})
	}
	return reqs, nil
}

// blank assignment to verify that ReconcileTenant implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTenant{}

// ReconcileTenant reconciles a Tenant object
type ReconcileTenant struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Tenant object and makes changes based on the state read
// and what is in the Tenant.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTenant) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Tenant")

	// Fetch the Tenant instance
	instance := &confiv1.Tenant{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		// Do Deletion Logic - right now we can just rely on owner references
		reqLogger.Info("Tenant instance marked for deletion, running finalizers")
		return reconcile.Result{}, r.runFinalizers(reqLogger, instance)
	}

	// Fetch the multitenancy object
	reqLogger.Info("Looking up tenancy kind")
	mt, err := instance.GetTenancy(r.client)
	if err != nil {
		return reconcile.Result{}, err
	}

	if mt.IsPaused() {
		reqLogger.Info("Multitenancy object is in paused state, skipping rest of the reconcile loop")
		return reconcile.Result{}, nil
	}

	// reconcile pod and configmap for tenant
	if err := r.reconcileItemsForResource(reqLogger, mt, instance); err != nil {
		return reconcile.Result{}, err
	}

	// ensure a tenace label on the tenant object for lookup purposes
	if err := r.ensureTenantLabels(reqLogger, mt, instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.ensureTenantFinalizer(reqLogger, instance)
}

// ensureTenantLabelsAndOwner merges the tenants user-defined labels with those
// used internally, and then pushes any necessary changes to the api servers.
func (r *ReconcileTenant) ensureTenantLabels(reqLogger logr.Logger, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	requiresUpdate := false
	labels := util.TenantLabels(mt, tenant, tenant.GetLabels())
	if reflect.DeepEqual(labels, tenant.GetLabels()) {
		reqLogger.Info("Tenant labels are up to date")
	} else {
		reqLogger.Info("Tenant labels need to be updated with multitenancy references")
		tenant.SetLabels(labels)
		requiresUpdate = true
	}
	if requiresUpdate {
		if err := r.client.Update(context.TODO(), tenant); err != nil {
			return err
		}
		return r.client.Get(context.TODO(), tenant.NamespacedName(), tenant)
	}
	return nil
}

// createItems returns all the items we would create if this were a new deployment
// these items should later be compared with the actual state to reconcile
func (r *ReconcileTenant) reconcileItemsForResource(reqLogger logr.Logger, mt *confiv1.MultiTenancy, tenant *confiv1.Tenant) error {
	configMap, err := resources.NewConfigMapForTenant(mt, tenant)
	if err != nil {
		return err
	}

	recreatedConfig, err := confireconcile.ReconcileConfigMap(r.client, reqLogger, mt, configMap)
	if err != nil {
		return err
	}

	// Create a Pod for the workload
	pod, err := resources.NewPodForTenant(mt, tenant)
	if err != nil {
		return err
	}
	if err = confireconcile.ReconcilePod(r.client, reqLogger, mt, pod, recreatedConfig); err != nil {
		return err
	}

	return nil
}

// ensureTenantFinalizer ensures that a finalizer is attached to the given tenant instance
func (r *ReconcileTenant) ensureTenantFinalizer(reqLogger logr.Logger, tenant *confiv1.Tenant) error {
	if util.StringSliceContains(tenant.GetFinalizers(), confiv1.TenantFinalizer) {
		reqLogger.Info("Tenant instance already contains finalizer")
		return nil
	}
	reqLogger.Info("Updating tenant instance with finalizer")
	tenant.SetFinalizers(append(tenant.GetFinalizers(), confiv1.TenantFinalizer))
	return r.client.Update(context.TODO(), tenant)
}

// removeFinalizer removes the finalizer from a tenant instance
func (r *ReconcileTenant) removeFinalizer(reqLogger logr.Logger, tenant *confiv1.Tenant) error {
	reqLogger.Info("Removing finalizer from tenant instance")
	tenant.SetFinalizers(util.StringSliceRemove(tenant.GetFinalizers(), confiv1.TenantFinalizer))
	return r.client.Update(context.TODO(), tenant)
}

// runFinalizers runs cleanup logic for a deleted tenant instance
func (r *ReconcileTenant) runFinalizers(reqLogger logr.Logger, tenant *confiv1.Tenant) error {
	if !util.StringSliceContains(tenant.GetFinalizers(), confiv1.TenantFinalizer) {
		reqLogger.Info("Finalizer has already been removed, assuming we are done here")
		return nil
	}

	mt, err := tenant.GetTenancy(r.client)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		reqLogger.Info("Parent multitenancy object appears to have been deleted, unable to check lifecycle hooks")
		return r.removeFinalizer(reqLogger, tenant)
	}

	for _, hook := range mt.GetDeleteHooks() {
		if err := eventhooks.ExecuteHook(confiv1.DeletedEvent, hook, mt, tenant); err != nil {
			return err
		}
	}

	return r.removeFinalizer(reqLogger, tenant)
}
