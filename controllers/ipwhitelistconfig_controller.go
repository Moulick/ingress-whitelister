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

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/corbaltcode/go-akamai"
	"github.com/corbaltcode/go-akamai/siteshield"
	"github.com/go-logr/logr"
	"inet.af/netaddr"

	corev1 "k8s.io/api/core/v1"
	knet "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	beta1 "github.com/Moulick/ingress-whitelister/api/v1beta1"
)

var ErrIPWhitelistConfigMissing = errors.New("no IPWhitelistConfig specified")

type ProviderString string

const (
	CloudflareProvider ProviderString = "cloudflare"
	AkamaiProvider     ProviderString = "akamai"
	FastlyProvider     ProviderString = "fastly"
	requeueInterval                   = 2 * time.Minute
	errRequeueInterval                = 5 * time.Second
)

// IPWhitelistConfigReconciler reconciles a IPWhitelistConfig object
type IPWhitelistConfigReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	IPWhitelistConfig string
	Log               logr.Logger
}

func (p ProviderString) String() string {
	return string(p)
}

// +kubebuilder:rbac:groups=ingress.security.moulick,resources=ipwhitelistconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.security.moulick,resources=ipwhitelistconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ingress.security.moulick,resources=ipwhitelistconfigs/finalizers,verbs=update

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
// Interesting thing to note here is that this Reconcile is triggered for Ingress Objects and not IPWhitelistConfig
func (r *IPWhitelistConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var finalWhiteList []string

	logo := r.Log.WithValues("ingress", req.NamespacedName)

	// check if IPWhitelistConfig is specified
	if r.IPWhitelistConfig == "" {
		logo.Error(ErrIPWhitelistConfigMissing, "Failed to reconcile as ErrIPWhitelistConfigMissing")

		return ctrl.Result{RequeueAfter: errRequeueInterval}, ErrIPWhitelistConfigMissing
	}
	logo.Info("Reconciling Ingress")
	// Fetch the IPWhitelistConfig instance
	ipWhitelistConfig, err := r.getIPWhitelistConfig(ctx)
	if err != nil {
		return ctrl.Result{RequeueAfter: errRequeueInterval}, err
	}

	// Fetch the IPWhitelistConfig ing
	ing := &knet.Ingress{}
	err = r.Get(ctx, req.NamespacedName, ing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{RequeueAfter: errRequeueInterval}, err
		}
		// we can ignore not found error as requing the ingress will not help anyways
		return ctrl.Result{}, nil
	}
	// loop over all the rules and check if the labels match
	for _, rule := range ipWhitelistConfig.Spec.Rules {
		selector, err := metav1.LabelSelectorAsSelector(rule.Selector)
		if err != nil {
			logo.Error(err, "failed to convert the labelSelector to selector")
			return ctrl.Result{}, err
		}
		// check if the ingress matches the defined selectors
		if selector.Matches(labels.Set(ing.GetLabels())) {
			logo.Info("Ingress matches the rule", "rule", rule.Name)
			// Loop through the list of IPGroupSelector
			for _, ipGroup := range rule.IPGroupSelector {
				// Loop and check if the ipSets has any
				for _, group := range ipWhitelistConfig.Spec.IPGroups {
					if group.Name == ipGroup {
						now := metav1.Now()
						if !group.Expires.Before(&now) {
							logo.Info("ipGroup matched, added", "ipGroup", ipGroup)
							finalWhiteList = append(finalWhiteList, group.CIDRS...)
						} else {
							logo.Info("ipGroup matched but expired", "ipGroup", ipGroup, "expiry", group.Expires.Format(time.RFC1123))
						}
						// as soon as we find the matching group, we can break out of the loop
						break
					}
				}
			}

			for _, x := range rule.ProviderSelector {
				for _, y := range ipWhitelistConfig.Spec.Providers {
					if x.Name == y.Name {
						switch y.Type {
						case CloudflareProvider.String():
							logo.Info("Provider matched", "provider", y.Name)
							ips, err := getCloudFlareCidrs(y.Cloudflare)
							if err != nil {
								logo.Error(err, "failed to get cloudflare cidrs")
								return ctrl.Result{RequeueAfter: errRequeueInterval}, err
							}
							for _, ip := range ips {
								finalWhiteList = append(finalWhiteList, ip.String())
							}
						case AkamaiProvider.String():
							logo.Info("Provider matched", "provider", y.Name)
							ips, err := r.getAkamaiCidrs(ctx, y.Akamai)
							if err != nil {
								logo.Error(err, "failed to get akamai cidrs")
								// if we fail to get CIDRs from akami, we might want to slow down the reconciliation loop, the api call to akamai is slow
								return ctrl.Result{RequeueAfter: 15 * time.Second}, err
							}
							for _, ip := range ips {
								finalWhiteList = append(finalWhiteList, ip.String())
							}
						case FastlyProvider.String():
							logo.Info("Provider matched", "provider", y.Name)
							logo.Info("fastly provider not implemented yet")
						}
					}
				}
			}
			// also we can break out after matching the first rule
			break
		}

	}

	// if the finalWhiteList is empty, then no rule matched, so we can try to remove the annotation
	if len(finalWhiteList) == 0 {
		logo.Info("No rule matched, skipping and/or cleaning up")
		if ing.Annotations == nil {
			// if the finalWhiteList is empty and no rule matched, don't need to do anything
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		// if the annotations are not nil, we can try to delete the annotation
		var deleted bool
		if ing.Annotations, deleted = deleteAnnotation(ing.Annotations, ipWhitelistConfig.Spec.WhitelistAnnotation); deleted {
			if err = r.Update(ctx, ing); err != nil {
				logo.Error(err, "failed to update the ingress")
				return ctrl.Result{RequeueAfter: errRequeueInterval}, err
			}
			logo.Info("removed annotation from ingress")

			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		// there was nothing to delete or update, we are done here
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}
	// Here we have a whitelist and we might need to update the annotations
	// sort the finalWhiteList
	finalWhiteListString := strings.Join(uniqueSorted(finalWhiteList), ",")
	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}
	// we need to check if the value is same
	if !annotationAlreadyEqual(ing.Annotations, ipWhitelistConfig.Spec.WhitelistAnnotation, finalWhiteListString) {
		// if above condition is false, we need to update the annotation
		ing.Annotations[ipWhitelistConfig.Spec.WhitelistAnnotation] = finalWhiteListString
		if err = r.Update(ctx, ing); err != nil {
			logo.Error(err, "failed to update the ingress")
			return ctrl.Result{RequeueAfter: errRequeueInterval}, err
		}
		logo.Info("updated the ingress")
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	logo.Info("ingress already up-to-date")
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// getIPWhitelistConfig retrieves the ruleSet configuration.
func (r *IPWhitelistConfigReconciler) getIPWhitelistConfig(ctx context.Context) (*beta1.IPWhitelistConfig, error) {
	var iPWhitelistConfig beta1.IPWhitelistConfig
	err := r.Get(ctx, client.ObjectKey{Name: r.IPWhitelistConfig}, &iPWhitelistConfig)
	if err != nil {
		return nil, err
	}

	return &iPWhitelistConfig, nil
}

// Return a list of unique and sorted list of strings from input
func uniqueSorted(slice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i] < list[j]
	})

	return list
}

// deleteAnnotation from annotations is they exist, used for cleanup, will return true if the annotation was deleted
func deleteAnnotation(annotations map[string]string, anno string) (map[string]string, bool) {
	if _, ok := annotations[anno]; ok {
		delete(annotations, anno)
		return annotations, true
	}
	return annotations, false
}

// annotationAlreadyEqual compares the annotation value to the given value, will return true if the value is same
func annotationAlreadyEqual(annotations map[string]string, anno string, value string) bool {
	if _, ok := annotations[anno]; ok {
		// so annotation exists, return true if value is same
		return annotations[anno] == value
	}
	// if the annotation does not exist, return false
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPWhitelistConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: add predicate filtering https://sdk.operatorframework.io/docs/building-operators/golang/references/event-filtering/
	return ctrl.NewControllerManagedBy(mgr).
		// For(&beta1.IPWhitelistConfig{}).
		For(&knet.Ingress{}).
		Complete(r)
}

func (r *IPWhitelistConfigReconciler) readSecretKey(ctx context.Context, secretRef *beta1.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	if err := r.Get(
		ctx,
		types.NamespacedName{
			Namespace: secretRef.Secret.Namespace,
			Name:      secretRef.Secret.Name,
		},
		secret,
	); err != nil {
		return "", err
	}
	val, ok := secret.Data[secretRef.Key]
	if !ok {
		err := fmt.Errorf("secret missing key")
		r.Log.Error(err, "missing key in secret", "namespacedname", secret.Namespace+"/"+secret.Name, "key", secretRef.Key)
		return "", err
	}
	return string(val), nil
}

// getCloudFlareCidrs returns the CIDRs for the given CloudFlare provider.
// This code is copied from https://github.com/cloudflare/cloudflare-go/blob/master/ips.go and modified to take given URL as input.
func getCloudFlareCidrs(provider beta1.CloudflareProvider) ([]netaddr.IPPrefix, error) {
	var cloudFlareIps []netaddr.IPPrefix

	uri := provider.JsonApi
	resp, err := http.Get(uri) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to make http call to cloudflare: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	var cresp cloudflare.IPsResponse
	err = json.Unmarshal(body, &cresp)
	if err != nil {
		return nil, err
	}

	var ips cloudflare.IPRanges
	ips.IPv4CIDRs = cresp.Result.IPv4CIDRs
	ips.IPv6CIDRs = cresp.Result.IPv6CIDRs

	for _, ip := range cresp.Result.ChinaColos {
		if strings.Contains(ip, ":") {
			ips.ChinaIPv6CIDRs = append(ips.ChinaIPv6CIDRs, ip)
		} else {
			ips.ChinaIPv4CIDRs = append(ips.ChinaIPv4CIDRs, ip)
		}
	}

	for _, ip := range ips.IPv4CIDRs {
		parsedIPPrefix, err := netaddr.ParseIPPrefix(ip)
		if err != nil {
			return nil, err
		}
		cloudFlareIps = append(cloudFlareIps, parsedIPPrefix)
	}

	return cloudFlareIps, nil
}

func (r *IPWhitelistConfigReconciler) getAkamaiCidrs(ctx context.Context, provider beta1.AkamaiProvider) ([]netaddr.IPPrefix, error) {
	akaClient, err := r.getsiteShieldClient(ctx, provider)
	if err != nil {
		return nil, err
	}
	siteMap, err := akaClient.GetMap(provider.MapId.IntValue())
	if err != nil {
		return nil, err
	}
	if siteMap.ProposedCIDRs == nil || len(siteMap.ProposedCIDRs) == 0 {
		return siteMap.CurrentCIDRs, nil
	}
	return siteMap.ProposedCIDRs, nil
}

func (r *IPWhitelistConfigReconciler) getsiteShieldClient(ctx context.Context, provider beta1.AkamaiProvider) (*siteshield.Client, error) {
	var host string
	var clientSecret string
	var clientToken string
	var accessToken string
	var err error

	if provider.Host != nil {
		if host, err = r.readSecretKey(ctx, provider.Host); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("host is required")
	}
	if provider.ClientToken != nil {
		if clientToken, err = r.readSecretKey(ctx, provider.ClientToken); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("client token is required")
	}
	if provider.ClientSecret != nil {
		if clientSecret, err = r.readSecretKey(ctx, provider.ClientSecret); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("client secret is required")
	}
	if provider.AccessToken != nil {
		if accessToken, err = r.readSecretKey(ctx, provider.AccessToken); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("access token is required")
	}
	cred := akamai.Credentials{
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
		ClientToken:  clientToken,
		Host:         host,
	}

	shieldClient := siteshield.Client{Credentials: cred}
	return &shieldClient, nil
}
