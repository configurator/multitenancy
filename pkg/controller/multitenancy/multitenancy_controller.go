package multitenancy

import (
	"context"
	"time"

	confiv1 "github.com/configurator/multitenancy/pkg/apis/confi/v1"
	"github.com/configurator/multitenancy/pkg/garbagecollection"
	resources "github.com/configurator/multitenancy/pkg/resources"
	"github.com/configurator/multitenancy/pkg/util"
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

var log = logf.Log.WithName("controller_multitenancy")

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

	// Watch for changes to dependent tenant resources (owner references ensured by Tenant controller)
	err = c.Watch(&source.Kind{Type: &confiv1.Tenant{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &confiv1.MultiTenancy{},
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

// Reconcile reads that state of the cluster for a MultiTenancy object and makes changes based on the state read
// and what is in the MultiTenancy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMultiTenancy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling status for multitenancy")
	// Fetch the MultiTenancy instance
	instance := &confiv1.MultiTenancy{}
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

	if instance.IsPaused() {
		reqLogger.Info("Multitenancy object is in paused state, skipping rest of the reconcile loop")
		return reconcile.Result{}, nil
	}

	// Fetch tenants for this object
	reqLogger.Info("Fetching tenants for multitenancy object")
	tenants, err := getTenantsForMultiTenancy(r.client, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Do a status update for the multitenancy object
	reqLogger.Info("Determining current multitenancy status")
	newStatus, err := getMultiTenancyStatus(r.client, instance, tenants)
	if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info("Current status", "Status", newStatus)
	// Only update the status if it has changed - otherwise we get caught in a reconcile loop
	if !newStatus.EqualTo(&instance.Status) {
		reqLogger.Info("Publishing status update")
		instance.Status = newStatus
		if err = r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	var res reconcile.Result
	if newStatus.AvailableReplicas != newStatus.Replicas {
		// let's make sure we check again in a bit - though a pod becoming available will also requeue us.
		// so this is really just an extra safety belt.
		reqLogger.Info("Not all desired replicas are available. Requeuing a status check in 5 seconds.")
		res = reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Duration(5) * time.Second,
		}
	} else {
		res = reconcile.Result{}
	}

	// Run the garbage collection and if nothing goes wrong go ahead and either requeue for a status change
	// or end the loop
	reqLogger.Info("Running garbage collection")
	return res, garbagecollection.RunGC(r.client)
}

// getTenantsForMultiTenancy returns all the tenants for a given multitenancy object
func getTenantsForMultiTenancy(c client.Client, mt *confiv1.MultiTenancy) ([]confiv1.Tenant, error) {
	tenants := &confiv1.TenantList{}
	return tenants.Items, c.List(context.TODO(), tenants, mt.NamespaceSelector(), mt.Selector())
}

// getMultiTenancyStatus iterates the pods deriving from a multitenancy object
// and returns a status object representing their state.
func getMultiTenancyStatus(c client.Client, mt *confiv1.MultiTenancy, tenants []confiv1.Tenant) (confiv1.MultiTenancyStatus, error) {
	status := confiv1.MultiTenancyStatus{}
	status.ObservedGeneration = mt.GetGeneration()
	status.Replicas = int32(len(tenants))
	for _, tenant := range tenants {
		pod, err := resources.GetPodForTenant(c, &tenant)
		if err != nil {
			return status, err
		}
		if util.PodIsReady(pod) {
			status.ReadyReplicas++
			// Let's assume available too for now
			status.AvailableReplicas++
		} else {
			status.UnavailableReplicas++
		}
		expected, err := resources.NewPodForTenant(mt, &tenant)
		if err != nil {
			return status, err
		}
		if util.AnnotationCreationSpecsEqual(pod.Annotations, expected.Annotations) {
			status.UpdatedReplicas++
		} else {
			status.OutdatedReplicas++
		}
	}
	return status, nil
}
