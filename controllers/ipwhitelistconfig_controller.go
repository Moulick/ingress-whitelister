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
	"errors"
	"net"
	"sort"
	"strings"
	"time"

	knet "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingresssecurityv1beta1 "github.com/Moulick/ingress-whitelister/api/v1beta1"
)

var (
	IPWhitelistConfigMissing = errors.New("no IPWhitelistConfig specified")
)

// IPWhitelistConfigReconciler reconciles a IPWhitelistConfig object
type IPWhitelistConfigReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	IPWhitelistConfig string
}

//+kubebuilder:rbac:groups=ingress.security.moulick,resources=ipwhitelistconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.security.moulick,resources=ipwhitelistconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.security.moulick,resources=ipwhitelistconfigs/finalizers,verbs=update

//+kubebuilder:rbac:groups=networking.k8s.io,resources=Ingress,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IPWhitelistConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IPWhitelistConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var finalWhiteList []string

	logo := log.FromContext(ctx).WithValues("ingress", req.NamespacedName)

	if r.IPWhitelistConfig == "" {
		logo.Error(IPWhitelistConfigMissing, "Failed to reconcile as IPWhitelistConfigMissing")
		return ctrl.Result{}, IPWhitelistConfigMissing
	}

	logo.Info("Reconciling Ingress")

	ruleSet, err := r.getRuleSet(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the IPWhitelistConfig ing
	ing := &knet.Ingress{}
	err = r.Get(ctx, req.NamespacedName, ing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, rule := range ruleSet.Spec.Rules {
		selector, err := v1.LabelSelectorAsSelector(rule.Selector)
		if err != nil {
			logo.Error(err, "failed to convert the labelSelector to selector")
			return ctrl.Result{}, err
		}
		// check if the ingress matches the defined selectors
		if selector.Matches(labels.Set(ing.GetLabels())) {
			// Loop through the list of IPGroupSelector
			for _, ipGroup := range rule.IPGroupSelector {
				// Loop and check if the ipSets has any
				for _, group := range ruleSet.Spec.IPGroup {
					if group.Name == ipGroup {
						now := v1.Now()
						if !group.Expires.Before(&now) {
							finalWhiteList = append(finalWhiteList, group.Cidrs...)
						}
						// as soon as we find the matching group, we can break out of the loop
						break
					}
				}
			}
		}
	}
	finalWhiteListString := strings.Join(uniqueSorted(finalWhiteList), ",")
	//logo.Info("Whitelist", "cidr", finalWhiteListString)

	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}

	ing.Annotations[ruleSet.Spec.WhitelistAnnotation] = finalWhiteListString

	if err = r.Update(ctx, ing); err != nil {
		logo.Error(err, "failed to update the ingress")
		return ctrl.Result{Requeue: true, RequeueAfter: 2 * time.Second}, err
	}

	// TODO: add removal of annotation if no label on ingress

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPWhitelistConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: add predicate filtering https://sdk.operatorframework.io/docs/building-operators/golang/references/event-filtering/

	return ctrl.NewControllerManagedBy(mgr).
		//For(&ingresssecurityv1beta1.IPWhitelistConfig{}).
		For(&knet.Ingress{}).
		Complete(r)
}

// getRuleSet retrieves the ruleSet configuration.
func (r *IPWhitelistConfigReconciler) getRuleSet(ctx context.Context) (*ingresssecurityv1beta1.IPWhitelistConfig, error) {
	var ruleSet ingresssecurityv1beta1.IPWhitelistConfig
	err := r.Get(ctx, client.ObjectKey{Name: r.IPWhitelistConfig}, &ruleSet)
	if err != nil {
		return nil, err
	}

	return &ruleSet, nil
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

// DeleteAnnotation from annotations is they exist, used for cleanup, no return
func DeleteAnnotation(annotations map[string]string, anno string) {
	if _, ok := annotations[anno]; ok {
		delete(annotations, anno)
	}
}

func checkIPAddress(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	return true
}
