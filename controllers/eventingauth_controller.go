/*
Copyright 2023.

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
	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	operatorv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
)

const requeueAfterError = time.Minute * 1

// eventingAuthReconciler reconciles a EventingAuth object
type eventingAuthReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	IasClient ias.Client
}

func NewEventingAuthReconciler(c client.Client, s *runtime.Scheme, ias ias.Client) ManagedReconciler {
	return &eventingAuthReconciler{
		Client:    c,
		Scheme:    s,
		IasClient: ias,
	}
}

// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/finalizers,verbs=update
func (r *eventingAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling EventingAuth", "name", req.Name, "namespace", req.Namespace)

	cr, err := fetchEventingAuth(ctx, r.Client, req.NamespacedName)
	if err != nil {
		logger.Info("EventingAuth not found", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	_, err = r.IasClient.CreateApplication("dummy")
	if err != nil {
		logger.Error(err, "Failed to create IAS application", "eventingAuth", cr.Name, "eventingAuthNamespace", cr.Namespace)
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, err
	}

	if err := updateStatus(ctx, r.Client, cr, operatorv1alpha1.StateOk); err != nil {
		logger.Error(err, "Failed to update status of EventingAuth", "name", cr.Name, "namespace", cr.Namespace)
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, err
	}

	return ctrl.Result{}, nil
}

func fetchEventingAuth(ctx context.Context, c client.Client, name types.NamespacedName) (operatorv1alpha1.EventingAuth, error) {
	var cr operatorv1alpha1.EventingAuth
	err := c.Get(ctx, name, &cr)
	if err != nil {
		return cr, err
	}
	return cr, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *eventingAuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.EventingAuth{}).
		Complete(r)
}

type ManagedReconciler interface {
	SetupWithManager(mgr ctrl.Manager) error
}
