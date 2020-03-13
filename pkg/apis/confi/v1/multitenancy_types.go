package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiTenancySpec defines the desired state of MultiTenancy
type MultiTenancySpec struct {
	// Which resource type we're replicating over. For each item of this resource type, the multitenancy
	// will create a pod.
	TenancyKind string `json:"tenancyKind"`

	// An envrionment variable name to be used so the pod knows which resource instance it's replicated for.
	// +optional
	TenantNameVariable string `json:"tenantNameVariable,omitempty"`

	// A volume name to be used so the pod knows which resource instance it's replicated for.
	// +optional
	TenantResourceVolume string `json:"tenantResourceVolume,omitempty"`

	// Indicates that the MultiTenancy is paused and will not be processed by the
	// resourceful set controller.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// A list of hooks to process during tenant lifecycle events
	// +optional
	EventHooks []EventHook `json:"eventHooks,omitempty"`

	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this deployment.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Template describes the pods that will be created.
	Template v1.PodTemplateSpec `json:"template"`
}

// EventHook represents an action to perform during a tenant's lifecycle
type EventHook struct {
	// A list of lifecycle events that trigger the hook
	Events []LifecycleEvent `json:"events"`

	// A log configuration to match when running event hooks (only applies to delete for now)
	// +optional
	LogConfig *LogConfig `json:"logConfig,omitempty"`

	// A slack config to process during the lifecycle event
	// +optional
	Slack *SlackConfig `json:"slack,omitempty"`
}

// LogConfig is used to represent a log parsing configuration for lifecycle hooks
type LogConfig struct {
	// The number of lines to tail when parsing tenant logs (defaults to 10)
	// +optional
	TailLines int64 `json:"tail,omitempty"`

	// A regex to match in the tailed log lines. If the regex does not produce a match
	// the hook will not fire.
	// +optional
	Regex string `json:"regex,omitempty"`

	// The container to tail logs from, defaults to only container in pods with
	// one container. Required if pod contains multiple containers.
	// +optional
	Container string `json:"container,omitempty"`
}

// SlackConfig represents a slack webhook configuration for lifecycle events
type SlackConfig struct {
	// The webhook URL to post slack messages to
	WebhookURL string `json:"webhookURL"`
}

// MultiTenancyStatus defines the observed state of MultiTenancy
type MultiTenancyStatus struct {
	// The generation observed by the deployment controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Total number of non-terminated pods targeted by this deployment that have the desired template spec.
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// Total number of non-terminated pods targeted by this deployment that do not have the desired template spec.
	// +optional
	OutdatedReplicas int32 `json:"outdatedReplicas,omitempty"`

	// Total number of ready pods targeted by this deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Total number of available pods (ready for at least minReadySeconds) targeted by this deployment.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Total number of unavailable pods targeted by this deployment. This is the total number of
	// pods that are still required for the deployment to have 100% available capacity. They may
	// either be pods that are running but not yet available or pods that still have not been created.
	// +optional
	UnavailableReplicas int32 `json:"unavailableReplicas,omitempty"`
}

// MultiTenancy is the Schema for the multitenancys API
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +k8s:openapi-gen=true
type MultiTenancy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiTenancySpec   `json:"spec,omitempty"`
	Status MultiTenancyStatus `json:"status,omitempty"`
}

// MultiTenancyList contains a list of MultiTenancy
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MultiTenancyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiTenancy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiTenancy{}, &MultiTenancyList{})
}
