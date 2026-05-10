// Package v1alpha1 contains the v1alpha1 API group types for the gforce operator.
//
// +groupName=gforce.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is the API group/version used to register these types.
	GroupVersion = schema.GroupVersion{Group: "gforce.io", Version: "v1alpha1"}

	// SchemeBuilder registers types with a runtime.Scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme registers all types in this package with a runtime.Scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
