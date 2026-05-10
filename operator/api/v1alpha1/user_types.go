package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GForceUserPhase is the lifecycle phase of a GForceUser resource.
type GForceUserPhase string

const (
	// GForceUserPhaseActive means the user account is enabled and in sync with the DB.
	GForceUserPhaseActive GForceUserPhase = "Active"
	// GForceUserPhaseSuspended means the user account has been administratively suspended.
	GForceUserPhaseSuspended GForceUserPhase = "Suspended"
)

// GForceUserSpec defines the desired state of a GForceUser.
type GForceUserSpec struct {
	// Username is the unique login handle for the user.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`
	// +kubebuilder:validation:MaxLength=39
	Username string `json:"username"`

	// Email is the user's primary email address.
	// +kubebuilder:validation:Required
	Email string `json:"email"`

	// DisplayName is the optional human-friendly name shown in the UI.
	// +kubebuilder:validation:MaxLength=100
	DisplayName string `json:"displayName,omitempty"`

	// IsAdmin grants platform-level administrative privileges.
	IsAdmin bool `json:"isAdmin,omitempty"`
}

// GForceUserStatus reports the observed state of a GForceUser.
type GForceUserStatus struct {
	// Phase is the current lifecycle phase of the user account.
	// +kubebuilder:validation:Enum=Active;Suspended
	Phase GForceUserPhase `json:"phase,omitempty"`

	// DatabaseID is the UUID of the corresponding record in the GForce database.
	DatabaseID string `json:"databaseID,omitempty"`

	// ObservedGeneration is the metadata.generation this status was computed from.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions follows standard Kubernetes condition conventions.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GForceUser is a Kubernetes CRD representing a GForce platform user account.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=gfuser,categories=gforce
// +kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`
// +kubebuilder:printcolumn:name="Email",type=string,JSONPath=`.spec.email`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type GForceUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GForceUserSpec   `json:"spec,omitempty"`
	Status GForceUserStatus `json:"status,omitempty"`
}

// GForceUserList contains a list of GForceUser resources.
//
// +kubebuilder:object:root=true
type GForceUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GForceUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GForceUser{}, &GForceUserList{})
}
