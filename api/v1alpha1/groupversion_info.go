/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package v1alpha1 contains API Schema definitions for the operator v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=operator.kyma-project.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// groupVersion is group version used to register these objects.
	groupVersion = schema.GroupVersion{Group: "operator.kyma-project.io", Version: "v1alpha1"} //nolint:gochecknoglobals // Used internally in the package.

	// schemeBuilder is used to add go types to the GroupVersionKind scheme.
	schemeBuilder = &scheme.Builder{GroupVersion: groupVersion} //nolint:gochecknoglobals // Used internally in the package.

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = schemeBuilder.AddToScheme //nolint:gochecknoglobals // Used outside the package.
)
