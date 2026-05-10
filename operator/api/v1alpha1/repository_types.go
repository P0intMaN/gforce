package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepositoryPhase is the lifecycle phase of a Repository resource.
// +kubebuilder:validation:Enum=Pending;Ready;Failed
type RepositoryPhase string

const (
	// RepositoryPhasePending means the repository has been accepted but not yet initialised on disk.
	RepositoryPhasePending RepositoryPhase = "Pending"
	// RepositoryPhaseReady means the bare git repository is initialised and serving.
	RepositoryPhaseReady RepositoryPhase = "Ready"
	// RepositoryPhaseFailed means the controller encountered an unrecoverable error.
	RepositoryPhaseFailed RepositoryPhase = "Failed"
)

// RepositorySpec defines the desired state of a Repository.
type RepositorySpec struct {
	// OwnerRef identifies the gforce User that owns this repository.
	// +kubebuilder:validation:Required
	OwnerRef string `json:"ownerRef"`

	// IsPrivate controls whether the repository is accessible only to its owner.
	// +kubebuilder:default=false
	IsPrivate bool `json:"isPrivate,omitempty"`

	// DefaultBranch is the branch that HEAD points to after initialisation.
	// +kubebuilder:default="main"
	// +kubebuilder:validation:MinLength=1
	DefaultBranch string `json:"defaultBranch,omitempty"`

	// Description is an optional human-readable summary of the repository's purpose.
	// +kubebuilder:validation:MaxLength=512
	Description string `json:"description,omitempty"`

	// StorageClass is the Kubernetes StorageClass to use when provisioning a
	// PersistentVolume for this repository. Defaults to the cluster default.
	// +optional
	StorageClass string `json:"storageClass,omitempty"`
}

// RepositoryStatus reports the observed state of a Repository.
type RepositoryStatus struct {
	// Phase is the lifecycle phase of the repository.
	Phase RepositoryPhase `json:"phase,omitempty"`

	// DiskPath is the absolute path on the storage volume where the bare git
	// repository lives. Set by the controller once the repo is initialised.
	DiskPath string `json:"diskPath,omitempty"`

	// Conditions represents the latest available observations of the repository's state.
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Repository is a Kubernetes CRD that represents a gforce-managed git repository.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=repo,categories=gforce
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=".spec.ownerRef"
// +kubebuilder:printcolumn:name="Private",type=boolean,JSONPath=".spec.isPrivate"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
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
