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

type ShowState string

const (
	ShowStateOpen     ShowState = "Open"
	ShowStateComplete ShowState = "Complete"
)

// ShowSpec defines the desired state of Show
type ShowSpec struct {
	// +required
	Capacity int `json:"capacity"`
}

// ShowStatus defines the observed state of Show
type ShowStatus struct {
	State              ShowState `json:"state,omitempty"`
	ObservedGeneration int       `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Show is the Schema for the shows API
type Show struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShowSpec   `json:"spec,omitempty"`
	Status ShowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ShowList contains a list of Show
type ShowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Show `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Show{}, &ShowList{})
}
