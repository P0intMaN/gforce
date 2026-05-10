// Package v1alpha1 contains the v1alpha1 API group types for the gforce operator.
//
// +groupName=gforce.dev
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is the group and version used to register the types in this package.
	GroupVersion = schema.GroupVersion{Group: "gforce.dev", Version: "v1alpha1"}

	// SchemeBuilder is used to add functions for the types in this package to a scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme registers the types in this package with a runtime.Scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
