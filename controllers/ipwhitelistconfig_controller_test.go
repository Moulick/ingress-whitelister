/*

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

package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"inet.af/netaddr"
	knet "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	// +kubebuilder:scaffold:imports

	beta1 "github.com/Moulick/ingress-whitelister/api/v1beta1"
)

const RuleSetName = "ipwhitelist-ruleset"

// +kubebuilder:docs-gen:collapse=Imports
// Define utility constants for object names and testing timeouts/durations and intervals.
const (
	Namespace          = "default"
	IngressServicePort = 8080

	ingressAdminName    = "ingress-admin"
	ingressAdminPath    = "/"
	ingressAdminHost    = "admin.example.com"
	ingressAdminService = "admin-service"

	ingressPublicName    = "ingress-public"
	ingressPublicPath    = "/"
	ingressPublicHost    = "comehackme.com"
	ingressPublicService = "example-service"

	ingressToolingName    = "ingress-tooling"
	ingressToolingPath    = "/"
	ingressToolingHost    = "grafana.website.com"
	ingressToolingService = "metrics-dashboard"

	whitelistLabel        = "ipwhitelist-type"
	whitelistAdminValue   = "admin"
	whitelistPublicValue  = "customerFacing"
	whitelistToolingValue = "tooling"

	SpecProviderCloudFlare = "cloudflare"
	SpecProviderFastly     = "fastly"
	SpecProviderAkamai     = "akamai"
	SpecProviderGitHub     = "github"

	timeout                   = time.Second * 20
	interval                  = time.Millisecond * 250
	randomWhitelistAnnotation = "random-whitelist-annotation"
)

var (
	pathType      = knet.PathTypeImplementationSpecific
	preservedAnno = map[string]string{"random-annotation": "should-have-been-preserved"}

	adminRule = beta1.Rule{
		Name: "admin",
		Selector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      whitelistLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						whitelistAdminValue,
					},
				},
			},
		},
		IPGroupSelector: []string{
			"admin",
			"devopsVPN",
			"siteA-vpn",
		},
		ProviderSelector: []beta1.ProviderSelector{
			{
				Name: SpecProviderCloudFlare,
			},
		},
	}
	internalRule = beta1.Rule{
		Name: "internal",
		Selector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      whitelistLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						whitelistToolingValue,
						"siteA-vpn",
					},
				},
			},
		},
		IPGroupSelector: []string{
			"admin",
			"devopsVPN",
		},
	}
	cfRule = beta1.Rule{
		Name: "public",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				whitelistLabel: whitelistPublicValue,
			},
		},
		ProviderSelector: []beta1.ProviderSelector{
			{
				Name: SpecProviderCloudFlare,
			},
		},
	}
	github = beta1.Rule{
		Name: "public",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				whitelistLabel: whitelistPublicValue,
			},
		},
		ProviderSelector: []beta1.ProviderSelector{
			{
				Name: SpecProviderGitHub,
			},
		},
	}
	devopsOnlyRule = beta1.Rule{
		Name: "devopsOnly",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				whitelistLabel: "devopsOnly",
			},
		},
		IPGroupSelector: []string{
			"devopsVPN",
		},
	}

	adminGroup = beta1.IPGroup{
		Name:    "admin",
		Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
		CIDRS:   []string{"192.169.0.1/32", "10.0.3.4/18"},
	}
	publicGroup = beta1.IPGroup{
		Name:    "public",
		Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
		CIDRS:   []string{"0.0.0.0/0", "::/0"},
	}
	devopsVPNGroup = beta1.IPGroup{
		Name:    "devopsVPN",
		Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
		CIDRS:   []string{"176.34.201.164/32"},
	}
	siteAGroup = beta1.IPGroup{
		Name:    "siteA-vpn",
		Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
		CIDRS: []string{
			"156.75.1.1/24",
		},
	}
)
var _ = Describe("IPWhitelistConfig controller Cloudflare", Ordered, func() {
	ipwhitelistrulesetCloudflare := beta1.IPWhitelistConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: beta1.GroupVersion.String(),
			Kind:       "IPWhitelistConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: RuleSetName,
		},
		Spec: beta1.IPWhitelistConfigSpec{
			WhitelistAnnotation: randomWhitelistAnnotation,
			Rules: []beta1.Rule{
				adminRule,
				internalRule,
				cfRule,
				devopsOnlyRule,
			},
			IPGroups: []beta1.IPGroup{
				adminGroup,
				publicGroup,
				devopsVPNGroup,
				siteAGroup,
			},
			Providers: []beta1.Providers{
				{
					Name: SpecProviderCloudFlare,
					Type: beta1.Cloudflare,

					Cloudflare: beta1.CloudflareProvider{
						JsonApi: "https://api.cloudflare.com/client/v4/ips",
					},
				},
			},
		},
	}

	ingressAdmin := knet.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ingressAdminName,
			Namespace:   Namespace,
			Annotations: preservedAnno,
			Labels: map[string]string{
				whitelistLabel: whitelistAdminValue,
				"random-label": "random-label-value",
			},
		},
		Spec: knet.IngressSpec{
			Rules: []knet.IngressRule{
				{
					Host: ingressAdminHost,
					IngressRuleValue: knet.IngressRuleValue{
						HTTP: &knet.HTTPIngressRuleValue{
							Paths: []knet.HTTPIngressPath{
								{
									Path:     ingressAdminPath,
									PathType: &pathType,
									Backend: knet.IngressBackend{
										Service: &knet.IngressServiceBackend{
											Name: ingressAdminService,
											Port: knet.ServiceBackendPort{
												Number: IngressServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ingressPublic := knet.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressPublicName,
			Namespace: Namespace,
			Labels: map[string]string{
				whitelistLabel: whitelistPublicValue,
			},
		},
		Spec: knet.IngressSpec{
			Rules: []knet.IngressRule{
				{
					Host: ingressPublicHost,
					IngressRuleValue: knet.IngressRuleValue{
						HTTP: &knet.HTTPIngressRuleValue{
							Paths: []knet.HTTPIngressPath{
								{
									Path:     ingressPublicPath,
									PathType: &pathType,
									Backend: knet.IngressBackend{
										Service: &knet.IngressServiceBackend{
											Name: ingressPublicService,
											Port: knet.ServiceBackendPort{
												Number: IngressServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ingressTooling := knet.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressToolingName,
			Namespace: Namespace,
			Labels: map[string]string{
				whitelistLabel: whitelistToolingValue,
			},
		},
		Spec: knet.IngressSpec{
			Rules: []knet.IngressRule{
				{
					Host: ingressToolingHost,
					IngressRuleValue: knet.IngressRuleValue{
						HTTP: &knet.HTTPIngressRuleValue{
							Paths: []knet.HTTPIngressPath{
								{
									Path:     ingressToolingPath,
									PathType: &pathType,
									Backend: knet.IngressBackend{
										Service: &knet.IngressServiceBackend{
											Name: ingressToolingService,
											Port: knet.ServiceBackendPort{
												Number: IngressServicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	Context("When Ingress have labels", func() {
		It("Should add the ipwhitelist annotation to the Ingress", func() {
			By("Creating a IPWhitelistConfig")
			Expect(k8sClient.Create(ctx, &ipwhitelistrulesetCloudflare)).Should(Succeed())

			ipwhitelistRulesetKey := types.NamespacedName{Name: ipwhitelistrulesetCloudflare.Name}
			createdipwhitelistRuleset := &beta1.IPWhitelistConfig{}
			// check creation
			Eventually(func(g Gomega) bool {
				err := k8sClient.Get(ctx, ipwhitelistRulesetKey, createdipwhitelistRuleset)
				g.Expect(err).ShouldNot(HaveOccurred())
				return true
			}, timeout, interval).Should(BeTrue())

			// Let's create and ingress with admin label
			By("Creating a admin ingress")
			Expect(k8sClient.Create(ctx, &ingressAdmin)).Should(Succeed())

			ingressAdminKey := types.NamespacedName{Name: ingressAdmin.Name, Namespace: ingressAdmin.Namespace}
			// check creation
			Eventually(func() bool {
				createdAdminIngress := &knet.Ingress{}
				err := k8sClient.Get(ctx, ingressAdminKey, createdAdminIngress)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Let's create an ingress that is completely public
			By("Creating a public ingress")
			Expect(k8sClient.Create(ctx, &ingressPublic)).Should(Succeed())

			ingressPublicKey := types.NamespacedName{Name: ingressPublic.Name, Namespace: ingressPublic.Namespace}
			// check creation
			Eventually(func() bool {
				createdPublicIngress := &knet.Ingress{}

				err := k8sClient.Get(ctx, ingressPublicKey, createdPublicIngress)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Let's create and ingress that is only supposed to be allowed to devops
			By("Creating a tooling ingress")
			Expect(k8sClient.Create(ctx, &ingressTooling)).Should(Succeed())

			ingressToolingKey := types.NamespacedName{Name: ingressTooling.Name, Namespace: ingressTooling.Namespace}
			// check creation
			Eventually(func() bool {
				createdToolingIngress := &knet.Ingress{}
				err := k8sClient.Get(ctx, ingressToolingKey, createdToolingIngress)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking that the admin has the ipwhitelist label")
			labeledAdminIngress := &knet.Ingress{}
			Eventually(func(g Gomega) map[string]string {
				err := k8sClient.Get(ctx, ingressAdminKey, labeledAdminIngress)
				g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
				return labeledAdminIngress.Annotations
			}, timeout, interval).Should(HaveKeyWithValue(randomWhitelistAnnotation, "10.0.3.4/18,103.21.244.0/22,103.22.200.0/22,103.31.4.0/22,104.16.0.0/13,104.24.0.0/14,108.162.192.0/18,131.0.72.0/22,141.101.64.0/18,156.75.1.1/24,162.158.0.0/15,172.64.0.0/13,173.245.48.0/20,176.34.201.164/32,188.114.96.0/20,190.93.240.0/20,192.169.0.1/32,197.234.240.0/22,198.41.128.0/17"))

			By("Checking that the Public has the ipwhitelist label")
			labeledPublicIngress := &knet.Ingress{}
			Eventually(func(g Gomega) map[string]string {
				err := k8sClient.Get(ctx, ingressPublicKey, labeledPublicIngress)
				g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
				return labeledPublicIngress.Annotations
			}, timeout, interval).Should(HaveKeyWithValue(randomWhitelistAnnotation, "103.21.244.0/22,103.22.200.0/22,103.31.4.0/22,104.16.0.0/13,104.24.0.0/14,108.162.192.0/18,131.0.72.0/22,141.101.64.0/18,162.158.0.0/15,172.64.0.0/13,173.245.48.0/20,188.114.96.0/20,190.93.240.0/20,197.234.240.0/22,198.41.128.0/17"))

			// Let's make sure our tooling ingress did not loose any other annotation after processing by controller // Keep at end of testing
			By("Checking that any other annotation should not be removed accidentally")
			processedAdminIngress := &knet.Ingress{}
			err := k8sClient.Get(ctx, ingressAdminKey, processedAdminIngress)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(processedAdminIngress.Annotations).Should(HaveKeyWithValue("random-annotation", preservedAnno["random-annotation"]), "any random annotation should be preserved")
		})

		Context("When label is removed from admin ingress", func() {
			It("Should remove the ipwhitelist annotation from the Ingress", func() {
				ingressAdminKey := types.NamespacedName{Name: ingressAdmin.Name, Namespace: ingressAdmin.Namespace}

				By("Removing label from the admin ingress")
				// Using eventually here because the controller and ginkgo both are updating the ingress and we need to handle the conflicting updates
				Eventually(func() bool {
					By("Getting the current admin ingress with annotation")
					currentAdminIngress := knet.Ingress{}

					Eventually(func(g Gomega) map[string]string {
						err := k8sClient.Get(ctx, ingressAdminKey, &currentAdminIngress)
						g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
						return currentAdminIngress.Annotations
					}, timeout, interval).Should(HaveKey(randomWhitelistAnnotation))

					delete(currentAdminIngress.Labels, whitelistLabel)

					err := k8sClient.Update(ctx, &currentAdminIngress)
					if err != nil {
						GinkgoWriter.Println("error updating ingress", err)
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())

				By("Checking that the admin ingress does not have the ipwhitelist annotation")
				Eventually(func(g Gomega) map[string]string {
					labeledAdminIngress := &knet.Ingress{}
					err := k8sClient.Get(ctx, ingressAdminKey, labeledAdminIngress)
					g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
					return labeledAdminIngress.Annotations
				}, timeout, interval).ShouldNot(HaveKey(randomWhitelistAnnotation))
			})
		})
	})
})

// var _ = Describe("IPWhitelistConfig controller GitHub", Ordered, func() {
// 	ipwhitelistrulesetGitHub := beta1.IPWhitelistConfig{
// 		TypeMeta: metav1.TypeMeta{
// 			APIVersion: beta1.GroupVersion.String(),
// 			Kind:       "IPWhitelistConfig",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: RuleSetName,
// 		},
// 		Spec: beta1.IPWhitelistConfigSpec{
// 			WhitelistAnnotation: randomWhitelistAnnotation,
// 			Rules: []beta1.Rule{
// 				adminRule,
// 			},
// 			IPGroups: []beta1.IPGroup{
// 				adminGroup,
// 			},
// 			Providers: []beta1.Providers{
// 				{
// 					Name: SpecProviderGitHub,
// 					Type: beta1.Github,
//
// 					Github: beta1.GithubProvider{
// 						JsonApi:  "https://api.cloudflare.com/client/v4/ips",
// 						Services: []string{"hooks"},
// 					},
// 				},
// 			},
// 		},
// 	}
//
// 	// ingressAdmin := knet.Ingress{
// 	// 	ObjectMeta: metav1.ObjectMeta{
// 	// 		Name:        ingressAdminName,
// 	// 		Namespace:   Namespace,
// 	// 		Annotations: preservedAnno,
// 	// 		Labels: map[string]string{
// 	// 			whitelistLabel: whitelistAdminValue,
// 	// 			"random-label": "random-label-value",
// 	// 		},
// 	// 	},
// 	// 	Spec: knet.IngressSpec{
// 	// 		Rules: []knet.IngressRule{
// 	// 			{
// 	// 				Host: ingressAdminHost,
// 	// 				IngressRuleValue: knet.IngressRuleValue{
// 	// 					HTTP: &knet.HTTPIngressRuleValue{
// 	// 						Paths: []knet.HTTPIngressPath{
// 	// 							{
// 	// 								Path:     ingressAdminPath,
// 	// 								PathType: &pathType,
// 	// 								Backend: knet.IngressBackend{
// 	// 									Service: &knet.IngressServiceBackend{
// 	// 										Name: ingressAdminService,
// 	// 										Port: knet.ServiceBackendPort{
// 	// 											Number: IngressServicePort,
// 	// 										},
// 	// 									},
// 	// 								},
// 	// 							},
// 	// 						},
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 	},
// 	// }
//
// 	Context("When Ingress have labels", func() {
// 		It("Should add the ipwhitelist annotation to the Ingress", func() {
// 			By("Creating a IPWhitelistConfig")
// 			Expect(k8sClient.Create(ctx, &ipwhitelistrulesetGitHub)).Should(Succeed())
//
// 			ipwhitelistRulesetKey := types.NamespacedName{Name: ipwhitelistrulesetGitHub.Name}
// 			createdipwhitelistRuleset := &beta1.IPWhitelistConfig{}
// 			// check creation
// 			Eventually(func(g Gomega) bool {
// 				err := k8sClient.Get(ctx, ipwhitelistRulesetKey, createdipwhitelistRuleset)
// 				g.Expect(err).ShouldNot(HaveOccurred())
// 				return true
// 			}, timeout, interval).Should(BeTrue())
//
// 			// // Let's create and ingress with admin label
// 			// By("Creating a admin ingress")
// 			// Expect(k8sClient.Create(ctx, &ingressAdmin)).Should(Succeed())
// 			//
// 			// ingressAdminKey := types.NamespacedName{Name: ingressAdmin.Name, Namespace: ingressAdmin.Namespace}
// 			// // check creation
// 			// Eventually(func() bool {
// 			// 	createdAdminIngress := &knet.Ingress{}
// 			// 	err := k8sClient.Get(ctx, ingressAdminKey, createdAdminIngress)
// 			// 	return err == nil
// 			// }, timeout, interval).Should(BeTrue())
// 			//
// 			// // Let's create an ingress that is completely public
// 			// By("Creating a public ingress")
// 			// Expect(k8sClient.Create(ctx, &ingressPublic)).Should(Succeed())
// 			//
// 			// ingressPublicKey := types.NamespacedName{Name: ingressPublic.Name, Namespace: ingressPublic.Namespace}
// 			// // check creation
// 			// Eventually(func() bool {
// 			// 	createdPublicIngress := &knet.Ingress{}
// 			//
// 			// 	err := k8sClient.Get(ctx, ingressPublicKey, createdPublicIngress)
// 			// 	return err == nil
// 			// }, timeout, interval).Should(BeTrue())
// 			//
// 			// // Let's create and ingress that is only supposed to be allowed to devops
// 			// By("Creating a tooling ingress")
// 			// Expect(k8sClient.Create(ctx, &ingressTooling)).Should(Succeed())
// 			//
// 			// ingressToolingKey := types.NamespacedName{Name: ingressTooling.Name, Namespace: ingressTooling.Namespace}
// 			// // check creation
// 			// Eventually(func() bool {
// 			// 	createdToolingIngress := &knet.Ingress{}
// 			// 	err := k8sClient.Get(ctx, ingressToolingKey, createdToolingIngress)
// 			// 	return err == nil
// 			// }, timeout, interval).Should(BeTrue())
// 			//
// 			// By("Checking that the admin has the ipwhitelist label")
// 			// labeledAdminIngress := &knet.Ingress{}
// 			// Eventually(func(g Gomega) map[string]string {
// 			// 	err := k8sClient.Get(ctx, ingressAdminKey, labeledAdminIngress)
// 			// 	g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
// 			// 	return labeledAdminIngress.Annotations
// 			// }, timeout, interval).Should(HaveKeyWithValue(randomWhitelistAnnotation, "10.0.3.4/18,103.21.244.0/22,103.22.200.0/22,103.31.4.0/22,104.16.0.0/13,104.24.0.0/14,108.162.192.0/18,131.0.72.0/22,141.101.64.0/18,156.75.1.1/24,162.158.0.0/15,172.64.0.0/13,173.245.48.0/20,176.34.201.164/32,188.114.96.0/20,190.93.240.0/20,192.169.0.1/32,197.234.240.0/22,198.41.128.0/17"))
// 			//
// 			// By("Checking that the Public has the ipwhitelist label")
// 			// labeledPublicIngress := &knet.Ingress{}
// 			// Eventually(func(g Gomega) map[string]string {
// 			// 	err := k8sClient.Get(ctx, ingressPublicKey, labeledPublicIngress)
// 			// 	g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
// 			// 	return labeledPublicIngress.Annotations
// 			// }, timeout, interval).Should(HaveKeyWithValue(randomWhitelistAnnotation, "103.21.244.0/22,103.22.200.0/22,103.31.4.0/22,104.16.0.0/13,104.24.0.0/14,108.162.192.0/18,131.0.72.0/22,141.101.64.0/18,162.158.0.0/15,172.64.0.0/13,173.245.48.0/20,188.114.96.0/20,190.93.240.0/20,197.234.240.0/22,198.41.128.0/17"))
// 			//
// 			// // Let's make sure our tooling ingress did not loose any other annotation after processing by controller // Keep at end of testing
// 			// By("Checking that any other annotation should not be removed accidentally")
// 			// processedAdminIngress := &knet.Ingress{}
// 			// err := k8sClient.Get(ctx, ingressAdminKey, processedAdminIngress)
// 			// Expect(err).ShouldNot(HaveOccurred())
// 			// Expect(processedAdminIngress.Annotations).Should(HaveKeyWithValue("random-annotation", preservedAnno["random-annotation"]), "any random annotation should be preserved")
// 		})
//
// 		// Context("When label is removed from admin ingress", func() {
// 		// 	It("Should remove the ipwhitelist annotation from the Ingress", func() {
// 		// 		ingressAdminKey := types.NamespacedName{Name: ingressAdmin.Name, Namespace: ingressAdmin.Namespace}
// 		//
// 		// 		By("Removing label from the admin ingress")
// 		// 		// Using eventually here because the controller and ginkgo both are updating the ingress and we need to handle the conflicting updates
// 		// 		Eventually(func() bool {
// 		// 			By("Getting the current admin ingress with annotation")
// 		// 			currentAdminIngress := knet.Ingress{}
// 		//
// 		// 			Eventually(func(g Gomega) map[string]string {
// 		// 				err := k8sClient.Get(ctx, ingressAdminKey, &currentAdminIngress)
// 		// 				g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
// 		// 				return currentAdminIngress.Annotations
// 		// 			}, timeout, interval).Should(HaveKey(randomWhitelistAnnotation))
// 		//
// 		// 			delete(currentAdminIngress.Labels, whitelistLabel)
// 		//
// 		// 			err := k8sClient.Update(ctx, &currentAdminIngress)
// 		// 			if err != nil {
// 		// 				GinkgoWriter.Println("error updating ingress", err)
// 		// 				return false
// 		// 			}
// 		// 			return true
// 		// 		}, timeout, interval).Should(BeTrue())
// 		//
// 		// 		By("Checking that the admin ingress does not have the ipwhitelist annotation")
// 		// 		Eventually(func(g Gomega) map[string]string {
// 		// 			labeledAdminIngress := &knet.Ingress{}
// 		// 			err := k8sClient.Get(ctx, ingressAdminKey, labeledAdminIngress)
// 		// 			g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
// 		// 			return labeledAdminIngress.Annotations
// 		// 		}, timeout, interval).ShouldNot(HaveKey(randomWhitelistAnnotation))
// 		// 	})
// 		// })
// 	})
// })

var _ = Describe("CloudFlare Provider Test", func() {
	knownIPv4 := []string{
		"173.245.48.0/20",
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"141.101.64.0/18",
		"108.162.192.0/18",
		"190.93.240.0/20",
		"188.114.96.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
		"162.158.0.0/15",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"172.64.0.0/13",
		"131.0.72.0/22",
	}
	var ipv4 []netaddr.IPPrefix
	for _, ip := range knownIPv4 {
		parsedIPPrefix, err := netaddr.ParseIPPrefix(ip)
		Expect(err).ShouldNot(HaveOccurred())
		ipv4 = append(ipv4, parsedIPPrefix)
	}
	Context("When Get CIDR's from Cloudflare", func() {
		It("should return the correct CIDR's", func() {
			By("Getting the CIDR's from Cloudflare")
			prov := beta1.CloudflareProvider{
				JsonApi: "https://api.cloudflare.com/client/v4/ips",
			}
			got, err := getCloudFlareCidrs(prov)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(HaveLen(len(ipv4)))
			for _, cidr := range ipv4 {
				Expect(got).To(ContainElement(cidr))
			}
		})
	})
})

var _ = Describe("GitHub Provider Test", func() {
	knownIPv4 := []string{
		"192.30.252.0/22",
		"185.199.108.0/22",
		"140.82.112.0/20",
		"143.55.64.0/20",
		"2a0a:a440::/29",
		"2606:50c0::/32",
	}
	var ipv4 []netaddr.IPPrefix
	for _, ip := range knownIPv4 {
		parsedIPPrefix, err := netaddr.ParseIPPrefix(ip)
		Expect(err).ShouldNot(HaveOccurred())
		ipv4 = append(ipv4, parsedIPPrefix)
	}
	Context("When Get CIDR's from GitHub", func() {
		It("should return the correct CIDR's", func() {
			By("Getting the CIDR's from GitHub")
			prov := beta1.GithubProvider{
				JsonApi:  "https://api.github.com/meta",
				Services: []string{"hooks"},
			}
			got, err := getGitHubCidrs(prov)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(HaveLen(len(ipv4)))
			for _, cidr := range ipv4 {
				Expect(got).To(ContainElement(cidr))
			}
		})
	})
})

// TODO: The below test throws a dereference error
// var _ = Describe("Testing readSecretKey function", Label("secret"), func() {
//	const (
//		secretName      = "newsecret"
//		secretNamespace = "default"
//	)
//	var (
//		hostVal    = randomString(10, GinkgoRandomSeed()*1)
//		ctokenVal  = randomString(15, GinkgoRandomSeed()*2)
//		atokenVal  = randomString(15, GinkgoRandomSeed()*3)
//		cSecretVal = randomString(15, GinkgoRandomSeed()*3)
//		hostKey    = "host"
//		ctokenKey  = "ctoken"
//		atokenKey  = "atoken"
//		cSecretKey = "csecret"
//
//		r = IPWhitelistConfigReconciler{
//			Client:            k8sClient,
//			Scheme:            scheme.Scheme,
//			IPWhitelistConfig: RuleSetName,
//			Log:               ctrl.Log.WithName("Ginkgo"),
//		}
//
//		host = beta1.SecretKeySelector{
//			Secret: corev1.SecretReference{
//				Name:      secretName,
//				Namespace: secretNamespace,
//			},
//			Key: hostKey,
//		}
//		clientToken = beta1.SecretKeySelector{
//			Secret: corev1.SecretReference{
//				Name:      secretName,
//				Namespace: secretNamespace,
//			},
//			Key: ctokenKey,
//		}
//		clientSecret = beta1.SecretKeySelector{
//			Secret: corev1.SecretReference{
//				Name:      secretName,
//				Namespace: secretNamespace,
//			},
//			Key: cSecretKey,
//		}
//		accessToken = beta1.SecretKeySelector{
//			Secret: corev1.SecretReference{
//				Name:      secretName,
//				Namespace: secretNamespace,
//			},
//			Key: atokenKey,
//		}
//	)
//	Context("Creating a Secret", func() {
//		It("Should be created", func() {
//			By("Generating a secret and creating in k8s cluster")
//
//			secret := &corev1.Secret{
//				ObjectMeta: metav1.ObjectMeta{
//					Name:      secretName,
//					Namespace: secretNamespace,
//				},
//				StringData: map[string]string{
//					hostKey:    hostVal,
//					ctokenKey:  ctokenVal,
//					atokenKey:  atokenVal,
//					cSecretKey: cSecretVal,
//				},
//			}
//			err := k8sClient.Create(ctx, secret)
//			Expect(err).ToNot(HaveOccurred())
//
//			By("reading secret")
//			secretRead := corev1.Secret{}
//			err = k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, &secretRead)
//			Expect(err).ToNot(HaveOccurred())
//			Expect(secretRead.Data).ToNot(Equal(nil))
//			Expect(secretRead.Data).To(HaveLen(4))
//
//			By("checking secret values")
//			val, err := r.readSecretKey(ctx, &host)
//
//			Expect(err).ToNot(HaveOccurred())
//			Expect(val).To(Equal(hostVal))
//
//			val, err = r.readSecretKey(ctx, &clientToken)
//			Expect(err).ToNot(HaveOccurred())
//			Expect(val).To(Equal(ctokenVal))
//
//			val, err = r.readSecretKey(ctx, &accessToken)
//			Expect(err).ToNot(HaveOccurred())
//			Expect(val).To(Equal(atokenVal))
//
//			val, err = r.readSecretKey(ctx, &clientSecret)
//			Expect(err).ToNot(HaveOccurred())
//			Expect(val).To(Equal(cSecretVal))
//		})
//	})
// })
//
