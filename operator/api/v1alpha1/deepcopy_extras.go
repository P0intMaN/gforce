// deepcopy_extras.go contains DeepCopyInto implementations for nested types
// that controller-gen v0.15.0 does not emit automatically.
// These complement zz_generated.deepcopy.go and must be kept in sync when
// Status or Spec structs gain new slice/pointer/map fields.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeepCopyInto copies all fields of RepositorySpec into out.
func (in *RepositorySpec) DeepCopyInto(out *RepositorySpec) {
	*out = *in
	out.OwnerRef = in.OwnerRef
}

// DeepCopyInto copies all fields of RepositoryStatus into out.
func (in *RepositoryStatus) DeepCopyInto(out *RepositoryStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies all fields of GForceUserSpec into out.
func (in *GForceUserSpec) DeepCopyInto(out *GForceUserSpec) {
	*out = *in
}

// DeepCopyInto copies all fields of GForceUserStatus into out.
func (in *GForceUserStatus) DeepCopyInto(out *GForceUserStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies all fields of OwnerReference into out.
func (in *OwnerReference) DeepCopyInto(out *OwnerReference) {
	*out = *in
}
