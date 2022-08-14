/*
Copyright 2021.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ProviderRef struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=provider
	Kind string `json:"kind"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XIntOrString
	MapId intstr.IntOrString `json:"mapId,omitempty"`
}

// IPGroup is a group of IPs with a set expiration time
type IPGroup struct {
	//+kubebuilder:validation:Required
	Name string `json:"name"`
	//+kubebuilder:validation:Required
	Expires metav1.Time `json:"expires"`
	// TODO: add ip validation
	//+kubebuilder:validation:Optional
	CIDRS []string `json:"cidrs,omitempty"`
	//+kubebuilder:validation:Optional
	Provider ProviderRef `json:"providerRef,omitempty"`
}

// Rule is mapping of an IPGroup to a set of labels
type Rule struct {
	//+kubebuilder:validation:Required
	Name string `json:"name"`
	//+kubebuilder:validation:Required
	Selector *metav1.LabelSelector `json:"selector"`
	//+kubebuilder:validation:Required
	IPGroupSelector []string `json:"ipGroupSelector"`
}

// IPWhitelistConfigSpec defines the desired state of IPWhitelistConfig
type IPWhitelistConfigSpec struct {
	//+kubebuilder:validation:Required
	WhitelistAnnotation string `json:"whitelistAnnotation"`
	//+kubebuilder:validation:Required
	Rules []Rule `json:"rules"`
	// +listType=map
	// +listMapKey=name
	//+kubebuilder:validation:Required
	IPGroups []IPGroup `json:"ipGroups"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// IPWhitelistConfig is the Schema for the ipwhitelistconfigs API
type IPWhitelistConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IPWhitelistConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// IPWhitelistConfigList contains a list of IPWhitelistConfig
type IPWhitelistConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPWhitelistConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPWhitelistConfig{}, &IPWhitelistConfigList{})
}
