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

package v1alpha1

import kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type State string

// Valid EventingAuth States.
const (
	StateReady    State = "Ready"
	StateNotReady State = "NotReady"
)

// EventingAuthSpec defines the desired state of EventingAuth.
type EventingAuthSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// EventingAuthStatus defines the observed state of EventingAuth.
type EventingAuthStatus struct {
	// State signifies current state of CustomObject. Value
	// can be one of ("Ready", "NotReady").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Ready;NotReady
	State State `json:"state,omitempty"`

	// Application contains information about a created IAS application
	Application *IASApplication `json:"iasApplication,omitempty"`
	// AuthSecret contains information about created K8s secret
	AuthSecret *AuthSecret `json:"secret,omitempty"`

	//  Conditions associated with EventingAuthStatus.
	Conditions []kmetav1.Condition `json:"conditions,omitempty"`
}

type IASApplication struct {
	// Name of the application in IAS
	Name string `json:"name"`
	// Application ID in IAS
	UUID string `json:"uuid"`
}

type AuthSecret struct {
	// NamespacedName of the secret on the managed runtime cluster
	NamespacedName string `json:"namespacedName"`
	// Runtime ID of the cluster where the secret is created
	ClusterID string `json:"clusterId"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"

// EventingAuth is the Schema for the eventingauths API.
type EventingAuth struct {
	kmetav1.TypeMeta   `json:",inline"`
	kmetav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventingAuthSpec   `json:"spec,omitempty"`
	Status EventingAuthStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventingAuthList contains a list of EventingAuth.
type EventingAuthList struct {
	kmetav1.TypeMeta `json:",inline"`
	kmetav1.ListMeta `json:"metadata,omitempty"`
	Items            []EventingAuth `json:"items"`
}

func init() { //nolint:gochecknoinits // Used on the package level.
	schemeBuilder.Register(&EventingAuth{}, &EventingAuthList{})
}
