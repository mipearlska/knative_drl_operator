/*
Copyright 2023 mipearlska.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DRLScaleActionSpec defines the desired state of DRLScaleAction
type DRLScaleActionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of DRLScaleAction. Edit drlscaleaction_types.go to remove/update
	ServiceHouse_Resource    string `json:"servicehouse_resource,omitempty"`
	ServiceHouse_Concurrency string `json:"servicehouse_concurrency,omitempty"`
	ServiceHouse_Podcount    string `json:"servicehouse_podcount,omitempty"`
	ServiceSenti_Resource    string `json:"servicesenti_resource,omitempty"`
	ServiceSenti_Concurrency string `json:"servicesenti_concurrency,omitempty"`
	ServiceSenti_Podcount    string `json:"servicesenti_podcount,omitempty"`
	ServiceNumbr_Resource    string `json:"servicenumbr_resource,omitempty"`
	ServiceNumbr_Concurrency string `json:"servicenumbr_concurrency,omitempty"`
	ServiceNumbr_Podcount    string `json:"servicenumbr_podcount,omitempty"`
}

// DRLScaleActionStatus defines the observed state of DRLScaleAction
type DRLScaleActionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DRLScaleAction is the Schema for the drlscaleactions API
type DRLScaleAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DRLScaleActionSpec   `json:"spec,omitempty"`
	Status DRLScaleActionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DRLScaleActionList contains a list of DRLScaleAction
type DRLScaleActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DRLScaleAction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DRLScaleAction{}, &DRLScaleActionList{})
}
