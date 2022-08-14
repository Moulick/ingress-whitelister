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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretKeySelector struct {
	Secret v1.SecretReference `json:"secret"`
	Key    string             `json:"key"`
}

type AkamaiProvider struct {
	// +kubebuilder:validation:Required
	Host *SecretKeySelector `json:"serviceConsumerDomainRef"`
	// +kubebuilder:validation:Required
	ClientToken *SecretKeySelector `json:"clientTokenSecretRef"`
	// +kubebuilder:validation:Required
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef"`
	// +kubebuilder:validation:Required
	AccessToken *SecretKeySelector `json:"accessTokenSecretRef"`
}

type CloudflareProvider struct {
	// +kubebuilder:validation:Required
	JsonApi string `json:"jsonApi"`
}

type FastlyProvider struct {
	// +kubebuilder:validation:Required
	JsonApi *string `json:"jsonApi"`
}

// ProviderSpec defines the desired state of Provider
type ProviderSpec struct {
	// +kubebuilder:validation:Optional
	Akamai *AkamaiProvider `json:"akamai,omitempty"`
	// +kubebuilder:validation:Optional
	Cloudflare *CloudflareProvider `json:"cloudflare,omitempty"`
	// +kubebuilder:validation:Optional
	Fastly *FastlyProvider `json:"fastly,omitempty"`
}

// ProviderStatus defines the observed state of Provider
type ProviderStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Provider is the Schema for the providers API
type Provider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderSpec   `json:"spec,omitempty"`
	Status ProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProviderList contains a list of Provider
type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Provider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Provider{}, &ProviderList{})
}
