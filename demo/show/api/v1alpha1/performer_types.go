/*
Copyright 2023 The MCTest Authors.

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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PerformerSpec defines the desired state of Performer
type PerformerSpec struct {
	// +required
	Name string `json:"name,omitempty"`
}

// PerformerStatus defines the observed state of Performer
type PerformerStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Performer is the Schema for the performers API
type Performer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PerformerSpec   `json:"spec,omitempty"`
	Status PerformerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PerformerList contains a list of Performer
type PerformerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Performer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Performer{}, &PerformerList{})
}
