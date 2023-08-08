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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type SecretKeySelector struct {
	Secret v1.SecretReference `json:"secret"`
	Key    string             `json:"key"`
}

type AkamaiProvider struct {
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Optional
	MapId *intstr.IntOrString `json:"mapId,omitempty"`
	// +kubebuilder:validation:Optional
	Host *SecretKeySelector `json:"serviceConsumerDomainRef,omitempty"`
	// +kubebuilder:validation:Optional
	ClientToken *SecretKeySelector `json:"clientTokenSecretRef,omitempty"`
	// +kubebuilder:validation:Optional
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef,omitempty"`
	// +kubebuilder:validation:Optional
	AccessToken *SecretKeySelector `json:"accessTokenSecretRef,omitempty"`
}

type Providers struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=akamai;cloudflare;fastly
	Type string `json:"type"`
	// +kubebuilder:validation:Optional
	Akamai AkamaiProvider `json:"akamai,omitempty"`
	// +kubebuilder:validation:Optional
	Cloudflare CloudflareProvider `json:"cloudflare,omitempty"`
	// +kubebuilder:validation:Optional
	Fastly FastlyProvider `json:"fastly,omitempty"`
}

type CloudflareProvider struct {
	// +kubebuilder:validation:Required
	JsonApi string `json:"jsonApi"`
}

type FastlyProvider struct {
	// +kubebuilder:validation:Required
	JsonApi string `json:"jsonApi"`
}

// IPGroup is a group of IPs with a set expiration time
type IPGroup struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Expires metav1.Time `json:"expires"`
	// TODO: add ip validation
	// +kubebuilder:validation:Optional
	CIDRS []string `json:"cidrs,omitempty"`
	// +kubebuilder:validation:Optional
}

type ProviderSelector struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// Rule is mapping of an IPGroup to a set of labels
type Rule struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Selector *metav1.LabelSelector `json:"selector"`
	// +kubebuilder:validation:Optional
	IPGroupSelector []string `json:"ipGroupSelector,omitempty"`
	// +kubebuilder:validation:Optional
	// +listMapKey=name
	// +listType=map
	ProviderSelector []ProviderSelector `json:"providerSelector,omitempty"`
}

// IPWhitelistConfigSpec defines the desired state of IPWhitelistConfig
type IPWhitelistConfigSpec struct {
	// +kubebuilder:validation:Required
	WhitelistAnnotation string `json:"whitelistAnnotation"`
	// +kubebuilder:validation:Required
	Rules []Rule `json:"rules"`
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:Optional
	IPGroups []IPGroup `json:"ipGroups"`

	// +listMapKey=name
	// +listType=map
	// +kubebuilder:validation:Optional
	Providers []Providers `json:"providers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// IPWhitelistConfig is the Schema for the ipwhitelistconfigs API
type IPWhitelistConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IPWhitelistConfigSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// IPWhitelistConfigList contains a list of IPWhitelistConfig
type IPWhitelistConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPWhitelistConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPWhitelistConfig{}, &IPWhitelistConfigList{})
}
