package v1

// LifecycleEvent is defined as a type to signify intent
type LifecycleEvent string

const (
	// CreatedEvent matches the creation of a tenant
	CreatedEvent LifecycleEvent = "created"
	// UpdatedEvent matches the update of a tenant
	UpdatedEvent = "updated"
	// DeletedEvent matches the deletion of a tenant
	DeletedEvent = "deleted"
)

const (
	// Labels and annotations
	CreationSpecAnnotationKey = "confi.gurator.com/creation-spec"
	TenantLabel               = "confi.gurator.com/tenant"
	MultiTenancyLabel         = "confi.gurator.com/multitenancy"
	ManagedByLabel            = "confi.gurator.com/manged-by"
	TenancyKindLabel          = "confi.gurator.com/tenancy-kind"
	ManagedByLabelValue       = "multitenancy_controller"

	// Finalizers
	MultitenancyFinalizer = "multitenancies.confi.gurator.com"
	TenantFinalizer       = "tenants.confi.gurator.com"
)

// boolPointer returns a pointer to a bool
func boolPointer(b bool) *bool { return &b }

// int64Pointer returns a pointer to an int64
func int64Pointer(i int64) *int64 { return &i }

// getHooksByEvent will iterate a list of event hook configurations and
// return all that match the given event.
func getHooksByEvent(inhooks []EventHook, ev LifecycleEvent) []EventHook {
	hooks := make([]EventHook, 0)
	for _, hook := range inhooks {
		if eventsContains(hook.Events, ev) {
			hooks = append(hooks, hook)
		}
	}
	return hooks
}

// eventsContains determines if a list of lifecycle events contains a given event.
func eventsContains(sev []LifecycleEvent, ev LifecycleEvent) bool {
	for _, x := range sev {
		if x == ev {
			return true
		}
	}
	return false
}
