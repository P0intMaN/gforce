package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepositoryPhase is the lifecycle phase of a Repository resource.
//
// +kubebuilder:validation:Enum=Pending;Ready;Failed;Deleting
type RepositoryPhase string

const (
	// RepositoryPhasePending means the resource has been accepted but not yet initialised.
	RepositoryPhasePending RepositoryPhase = "Pending"
	// RepositoryPhaseReady means the bare git repository is initialised and in sync with the DB.
	RepositoryPhaseReady RepositoryPhase = "Ready"
	// RepositoryPhaseFailed means the controller encountered an unrecoverable error.
	RepositoryPhaseFailed RepositoryPhase = "Failed"
	// RepositoryPhaseDeleting means the deletion workflow is in progress.
	RepositoryPhaseDeleting RepositoryPhase = "Deleting"
)

// Condition type constants for Repository resources.
const (
	ConditionReady          = "Ready"
	ConditionDiskReady      = "DiskReady"
	ConditionDatabaseSynced = "DatabaseSynced"
)

// OwnerReference identifies the GForce user who owns a repository.
type OwnerReference struct {
	// Username is the GForce username of the owning user.
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// UserID is the UUID of the user in the GForce database.
	// +kubebuilder:validation:Required
	UserID string `json:"userID"`
}

// RepositorySpec defines the desired state of a Repository.
//
// +kubebuilder:validation:XValidation:rule="self.name == oldSelf.name",message="name is immutable after creation"
type RepositorySpec struct {
	// OwnerRef references the GForce user who owns this repository.
	// +kubebuilder:validation:Required
	OwnerRef OwnerReference `json:"ownerRef"`

	// Name is the repository name in slug format.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// Description is an optional human-readable description.
	// +kubebuilder:validation:MaxLength=500
	Description string `json:"description,omitempty"`

	// IsPrivate controls whether the repository is visible only to its owner.
	IsPrivate bool `json:"isPrivate,omitempty"`

	// DefaultBranch is the branch that HEAD points to.
	// +kubebuilder:default=main
	DefaultBranch string `json:"defaultBranch,omitempty"`

	// StorageClass is the Kubernetes StorageClass used when provisioning the PVC.
	// +kubebuilder:validation:Optional
	StorageClass string `json:"storageClass,omitempty"`

	// DiskPath is the absolute path on the storage volume.
	// Set by the operator on first reconcile; immutable thereafter.
	// +kubebuilder:validation:Optional
	DiskPath string `json:"diskPath,omitempty"`
}

// RepositoryStatus reports the observed state of a Repository.
type RepositoryStatus struct {
	// Phase is the current lifecycle phase.
	Phase RepositoryPhase `json:"phase,omitempty"`

	// DiskPath is the on-disk location of the bare git repository.
	DiskPath string `json:"diskPath,omitempty"`

	// DatabaseID is the UUID of the corresponding record in the GForce database.
	DatabaseID string `json:"databaseID,omitempty"`

	// ObservedGeneration is the metadata.generation this status was computed from.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions follows standard Kubernetes condition conventions.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Repository is a Kubernetes CRD representing a GForce-managed git repository.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=repo,categories=gforce
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.ownerRef.username`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// RepositoryList contains a list of Repository resources.
//
// +kubebuilder:object:root=true
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
