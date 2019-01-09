package v1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ResourcefulSetSpec defines the desired state of ResourcefulSet
type ResourcefulSetSpec struct {
	// Which resource type we're replicating over. For each item of this resource type, the resourcefulset
	// will create a pod.
	ReplicationForResource string `json:"replicateForResource"`

	// A volume name to be used so the pod knows which resource instance it's replicated for.
	// +optional
	ReplicationResourceVolume string `json:"replicationResourceVolume,omitempty"`

	// Indicates that the ResourcefulSet is paused and will not be processed by the
	// resourceful set controller.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this deployment.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Template describes the pods that will be created.
	Template v1.PodTemplateSpec `json:"template"`
}

// ResourcefulSetStatus defines the observed state of ResourcefulSet
type ResourcefulSetStatus struct {
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourcefulSet is the Schema for the resourcefulsets API
// +k8s:openapi-gen=true
type ResourcefulSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourcefulSetSpec   `json:"spec,omitempty"`
	Status ResourcefulSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourcefulSetList contains a list of ResourcefulSet
type ResourcefulSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourcefulSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourcefulSet{}, &ResourcefulSetList{})
}
