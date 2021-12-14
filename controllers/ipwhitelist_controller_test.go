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
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	knet "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	//+kubebuilder:scaffold:imports

	ingresssecurityv1beta1 "github.com/Moulick/ingress-whitelister/api/v1beta1"
)

// +kubebuilder:docs-gen:collapse=Imports

var pathType = knet.PathTypeImplementationSpecific
var preservedAnno = map[string]string{"random-annotation": "should-have-been-preserved"}
var _ = Describe("IPWhitelistConfig controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		Name               = "ipwhitelist-ruleset"
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

		timeout                   = time.Second * 10
		duration                  = time.Second * 10
		interval                  = time.Millisecond * 250
		randomWhitelistAnnotation = "random-whitelist-annotation"
	)

	Context("When Ingress have labels", func() {
		It("Should add the ipwhitelist annotation to the Ingress", func() {
			By("Creating a IPWhitelistConfig")
			ctx := context.Background()
			ipwhitelistRuleset := &ingresssecurityv1beta1.IPWhitelistConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ingresssecurityv1beta1.GroupVersion.String(),
					Kind:       "IPWhitelistConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: Name,
				},
				Spec: ingresssecurityv1beta1.IPWhitelistConfigSpec{
					WhitelistAnnotation: randomWhitelistAnnotation,
					Rules: []ingresssecurityv1beta1.Rule{
						{
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
						},
						{
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
						},
						{
							Name: "public",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									whitelistLabel: whitelistPublicValue,
								},
							},
							IPGroupSelector: []string{
								"public",
							},
						},
						{
							Name: "devopsOnly",
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									whitelistLabel: "devopsOnly",
								},
							},
							IPGroupSelector: []string{
								"devopsVPN",
							},
						},
					},
					IPGroup: []ingresssecurityv1beta1.IPGroup{
						{
							Name:    "admin",
							Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
							Cidrs: []string{
								"192.169.0.1/32",
								"10.0.3.4/18",
							},
						},
						{
							Name:    "public",
							Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
							Cidrs: []string{
								"0.0.0.0/0",
								"::/0",
							},
						},
						{
							Name:    "devopsVPN",
							Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
							Cidrs: []string{
								"176.34.201.164/32",
							},
						},
						{
							Name:    "siteA-vpn",
							Expires: metav1.Time{Time: time.Now().Add(2 * time.Hour)},
							Cidrs: []string{
								"156.75.1.1/24",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ipwhitelistRuleset)).Should(Succeed())

			ipwhitelistRulesetKey := types.NamespacedName{Name: ipwhitelistRuleset.Name}
			createdipwhitelistRuleset := &ingresssecurityv1beta1.IPWhitelistConfig{}
			// check creation
			Eventually(func(g Gomega) bool {
				err := k8sClient.Get(ctx, ipwhitelistRulesetKey, createdipwhitelistRuleset)
				g.Expect(err).ShouldNot(HaveOccurred())
				return true
			}, timeout, interval).Should(BeTrue())

			// Let's create and ingress with admin label
			By("Creating a admin ingress")
			ingressAdmin := &knet.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ingressAdminName,
					Namespace:   Namespace,
					Annotations: preservedAnno,
					Labels: map[string]string{
						whitelistLabel: whitelistAdminValue,
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
			Expect(k8sClient.Create(ctx, ingressAdmin)).Should(Succeed())

			ingressAdminKey := types.NamespacedName{Name: ingressAdmin.Name, Namespace: ingressAdmin.Namespace}
			// check creation
			Eventually(func() bool {
				createdAdminIngress := &knet.Ingress{}
				err := k8sClient.Get(ctx, ingressAdminKey, createdAdminIngress)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// Let's create an ingress that is completely public
			By("Creating a public ingress")
			ingressPublic := &knet.Ingress{
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
			Expect(k8sClient.Create(ctx, ingressPublic)).Should(Succeed())

			ingressPublicKey := types.NamespacedName{Name: ingressPublic.Name, Namespace: ingressPublic.Namespace}
			// check creation
			Eventually(func() bool {
				createdPublicIngress := &knet.Ingress{}

				err := k8sClient.Get(ctx, ingressPublicKey, createdPublicIngress)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// Let's create and ingress that is only supposed to be allowed to devops
			By("Creating a tooling ingress")
			ingressTooling := &knet.Ingress{
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
			Expect(k8sClient.Create(ctx, ingressTooling)).Should(Succeed())

			ingressToolingKey := types.NamespacedName{Name: ingressTooling.Name, Namespace: ingressTooling.Namespace}
			// check creation
			Eventually(func() bool {
				createdToolingIngress := &knet.Ingress{}
				err := k8sClient.Get(ctx, ingressToolingKey, createdToolingIngress)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Checking that the admin has the ipwhitelist label")
			labeledAdminIngress := &knet.Ingress{}
			Eventually(func(g Gomega) map[string]string {
				err := k8sClient.Get(ctx, ingressAdminKey, labeledAdminIngress)
				g.Expect(err).ShouldNot(HaveOccurred(), "Failed to get the ingress")
				//g.Expect(labeledAdminIngress.Annotations).Should(HaveKeyWithValue(randomWhitelistAnnotation, "10.0.3.4/18,156.75.1.1/24,176.34.201.164/32,192.169.0.1/32"))
				return labeledAdminIngress.Annotations
			}, timeout, interval).Should(HaveKeyWithValue(randomWhitelistAnnotation, "10.0.3.4/18,156.75.1.1/24,176.34.201.164/32,192.169.0.1/32"))

			// Let's make sure our tooling ingress did not loose any other annotation after processing by controller TODO: move to end of testing
			// TODO: find a way to move this k8sClient.Get() into the expect
			By("Checking that any other annotation should not be removed accidentally")
			processedAdminIngress := &knet.Ingress{}
			err := k8sClient.Get(ctx, ingressAdminKey, processedAdminIngress)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(processedAdminIngress.Annotations).Should(HaveKeyWithValue("random-annotation", preservedAnno["random-annotation"]), "any random annotation should be preserved")
		})
	})

})
