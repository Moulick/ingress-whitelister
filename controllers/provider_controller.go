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
	"fmt"

	"github.com/corbaltcode/go-akamai"
	"github.com/corbaltcode/go-akamai/siteshield"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingresssecurityv1alpha1 "github.com/Moulick/ingress-whitelister/api/v1alpha1"
)

// ProviderReconciler reconciles a Provider object
type ProviderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ingress.security.moulick,resources=providers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.security.moulick,resources=providers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.security.moulick,resources=providers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Provider object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logo := log.FromContext(ctx).WithValues("provider", req.NamespacedName)

	// Fetch the Provider instance
	provider := &ingresssecurityv1alpha1.Provider{}
	if err := r.Get(ctx, req.NamespacedName, provider); err != nil {
		logo.Error(err, "unable to fetch Provider")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logo.Info("Reconciling Provider")

	// Check provider type
	if provider.Spec.Akamai != nil {

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingresssecurityv1alpha1.Provider{}).
		Complete(r)
}

func (r *ProviderReconciler) readSecretKey(ctx context.Context, secretRef *ingresssecurityv1alpha1.SecretKeySelector) (string, error) {
	secret := &v1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: secretRef.Secret.Namespace, Name: secretRef.Secret.Name}, secret); err != nil {
		return "", err
	}
	val, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf("secret %s missing key %s", secret.Namespace+"/"+secret.Name, secretRef.Key)
	}
	return string(val), nil
}

func (r *ProviderReconciler) demo(ctx context.Context, provider *ingresssecurityv1alpha1.AkamaiProvider) error {
	var host string
	var clientSecret string
	var clientToken string
	var accessToken string
	var err error

	if provider.Host != nil {
		if host, err = r.readSecretKey(ctx, provider.Host); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("host is required")
	}
	if provider.ClientToken != nil {
		if clientToken, err = r.readSecretKey(ctx, provider.ClientToken); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("client token is required")
	}
	if provider.ClientSecret != nil {
		if clientSecret, err = r.readSecretKey(ctx, provider.ClientSecret); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("client secret is required")

	}
	if provider.AccessToken != nil {
		if accessToken, err = r.readSecretKey(ctx, provider.AccessToken); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("access token is required")

	}

	cred := akamai.Credentials{
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
		ClientToken:  clientToken,
		Host:         host,
	}

	client := siteshield.Client{Credentials: cred}
	maps, err := client.GetMap(1935454)
	if err != nil {
		fmt.Println(err)
	}
	
}
