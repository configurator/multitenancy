package v1

import (
	"fmt"
	"regexp"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsPaused returns true if this multitenancy object is in a paused state
func (mt *MultiTenancy) IsPaused() bool {
	return mt.Spec.Paused
}

// GetCombinedName returns the combined resource name for this multitenancy object
// and a deriving tenant.
func (mt *MultiTenancy) GetCombinedName(tenant *Tenant) string {
	return fmt.Sprintf("%s-%s", mt.GetName(), tenant.GetName())
}

// HasResourceVolume returns true if this multitenancy object has a resource volume
// configuration.
func (mt *MultiTenancy) HasResourceVolume() bool {
	return mt.Spec.TenantResourceVolume != ""
}

// HasNameVariable returns true if this multitenancy object has a name variable.
func (mt *MultiTenancy) HasNameVariable() bool {
	return mt.Spec.TenantNameVariable != ""
}

// Selector returns the multitenancy label selector for this resource
func (mt *MultiTenancy) Selector() client.MatchingLabels {
	return client.MatchingLabels{MultiTenancyLabel: mt.GetName()}
}

// GetTenantResourceVolume returns the tenant resource volume spec for the given tenant
func (mt *MultiTenancy) GetTenantResourceVolume(tenant *Tenant) v1.Volume {
	return v1.Volume{
		Name: mt.Spec.TenantResourceVolume,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: mt.GetCombinedName(tenant),
				},
			},
		},
	}
}

// GetTenantEnvVar returns the env var spec for a tenant's name variable
func (mt *MultiTenancy) GetTenantEnvVar(tenant *Tenant) v1.EnvVar {
	return v1.EnvVar{
		Name:  mt.Spec.TenantNameVariable,
		Value: tenant.GetName(),
	}
}

// GetPodSpec returns the pod spec for the given tenant
func (mt *MultiTenancy) GetPodSpec(tenant *Tenant) v1.PodSpec {
	spec := mt.Spec.Template.Spec.DeepCopy()

	if mt.HasResourceVolume() {
		// Add a volume mapping to a ConfigMap
		spec.Volumes = append(spec.Volumes, mt.GetTenantResourceVolume(tenant))
	}

	if mt.HasNameVariable() {
		// Add an environment variable mapping - to every container in the pod
		for i := range spec.Containers {
			spec.Containers[i].Env = append(spec.Containers[i].Env, mt.GetTenantEnvVar(tenant))
		}
	}
	return *spec
}

// GetPodAnnotations returns the pod annotations for this multitenancy object.
func (mt *MultiTenancy) GetPodAnnotations() map[string]string {
	return mt.Spec.Template.Annotations
}

// NamespaceSelector returns the namespace selecgtor for this multitenancy object
func (mt *MultiTenancy) NamespaceSelector() client.InNamespace {
	return client.InNamespace(mt.GetNamespace())
}

// GetCreateHooks returns the hooks that should be fired for a creation event
func (s *MultiTenancy) GetCreateHooks() []EventHook {
	return getHooksByEvent(s.Spec.EventHooks, CreatedEvent)
}

// GetUpdateHooks returns the hooks that should be fired for an update event
func (s *MultiTenancy) GetUpdateHooks() []EventHook {
	return getHooksByEvent(s.Spec.EventHooks, UpdatedEvent)
}

// GetDeleteHooks returns the hooks that should be fired for a delete event
func (s *MultiTenancy) GetDeleteHooks() []EventHook {
	return getHooksByEvent(s.Spec.EventHooks, DeletedEvent)
}

// EqualTo returns if this multitenancy status is fully equal to a given one.
func (s *MultiTenancyStatus) EqualTo(status *MultiTenancyStatus) bool {
	return s.AvailableReplicas == status.AvailableReplicas &&
		s.OutdatedReplicas == status.OutdatedReplicas &&
		s.ReadyReplicas == status.ReadyReplicas &&
		s.Replicas == status.Replicas &&
		s.UnavailableReplicas == status.UnavailableReplicas &&
		s.UpdatedReplicas == status.UpdatedReplicas
}

// GetTail returns the number of lines to tail based on the current configuration
func (l *LogConfig) GetTailLines() *int64 {
	if l.TailLines == 0 {
		return int64Pointer(10)
	}
	return &l.TailLines
}

// GetContainer returns the container to tail logs from (or an empty string
// to default to only container).
func (l *LogConfig) GetContainer() string {
	return l.Container
}

// GetRegex returns a compiled regex to use for searching logs, or nil if one is
// not provided
func (l *LogConfig) GetRegex() *regexp.Regexp {
	if l.Regex == "" {
		return nil
	}
	return regexp.MustCompile(l.Regex)
}
