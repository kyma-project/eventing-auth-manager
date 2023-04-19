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
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	operatorv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
)

const (
	requeueAfterError          = time.Minute * 1
	applicationSecretName      = "eventing-auth-application"
	applicationSecretNamespace = "kyma-system"
	eventingAuthFinalizerName  = "eventingauth.operator.kyma-project.io/finalizer"
)

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

// TODO: Check if conditions are correctly represented
// TODO: Implement finalizer to have correct deletion handling
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/finalizers,verbs=update
func (r *eventingAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling EventingAuth", "name", req.Name, "namespace", req.Namespace)

	cr, err := fetchEventingAuth(ctx, r.Client, req.NamespacedName)
	if err != nil {
		logger.Error(err, "failed to fetch EventingAuth")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// check DeletionTimestamp to determine if object is under deletion
	if cr.ObjectMeta.DeletionTimestamp.IsZero() {
		if err = r.addFinalizer(ctx, &cr); err != nil {
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, err
		}
	} else {
		if err = r.handleDeletion(ctx, &cr); err != nil {
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// TODO: Use correct pointing to the target cluster
	appSecretExists, err := hasTargetClusterApplicationSecret(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to retrieve secret state from target cluster", "eventingAuth", cr.Name, "eventingAuthNamespace", cr.Namespace)
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, err
	}

	if !appSecretExists {

		// TODO: Name of the IAS application should be taken from Kyma CR owner reference
		iasApplication, err := r.IasClient.CreateApplication(ctx, fmt.Sprintf("eventing-auth-manager-%s", uuid.New()))
		if err != nil {
			logger.Error(err, "Failed to create IAS application", "eventingAuth", cr.Name, "eventingAuthNamespace", cr.Namespace)
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, err
		}

		appSecret := iasApplication.ToSecret(applicationSecretName, applicationSecretNamespace)

		// TODO: Create secret on target cluster by reading the kubeconfig from the secret using the name of the Kyma CR owner reference
		err = r.Client.Create(ctx, &appSecret)
		if err != nil {
			logger.Error(err, "Failed to create application secret", "eventingAuth", cr.Name, "eventingAuthNamespace", cr.Namespace)
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, err
		}
	}

	if err := updateStatus(ctx, r.Client, cr, operatorv1alpha1.StateOk); err != nil {
		logger.Error(err, "Failed to update status of EventingAuth", "name", cr.Name, "namespace", cr.Namespace)
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, err
	}

	return ctrl.Result{}, nil
}

// Adds the finalizer if none exists
func (r *eventingAuthReconciler) addFinalizer(ctx context.Context, cr *operatorv1alpha1.EventingAuth) error {
	if !controllerutil.ContainsFinalizer(cr, eventingAuthFinalizerName) {
		controllerutil.AddFinalizer(cr, eventingAuthFinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return fmt.Errorf("failed to add finalizer: %v", err)
		}
	}
	return nil
}

// Deletes the secret and IAS app. Finally, removes the finalizer.
func (r *eventingAuthReconciler) handleDeletion(ctx context.Context, cr *operatorv1alpha1.EventingAuth) error {
	// The object is being deleted
	if controllerutil.ContainsFinalizer(cr, eventingAuthFinalizerName) {
		// delete k8s secret
		if err := deleteSecret(ctx, r.Client); err != nil {
			return fmt.Errorf("failed to delete secret: %v", err)
		}
		// delete IAS application clean-up
		if err := r.IasClient.DeleteApplication(ctx, cr.Name); err != nil {
			return fmt.Errorf("failed to delete IAS Application: %v", err)
		}
		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(cr, eventingAuthFinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return fmt.Errorf("failed to remove finalizer: %v", err)
		}
	}
	return nil
}

func fetchEventingAuth(ctx context.Context, c client.Client, name types.NamespacedName) (operatorv1alpha1.EventingAuth, error) {
	var cr operatorv1alpha1.EventingAuth
	err := c.Get(ctx, name, &cr)
	if err != nil {
		return cr, err
	}
	return cr, nil
}

func hasTargetClusterApplicationSecret(ctx context.Context, c client.Client) (bool, error) {
	var s v1.Secret
	err := c.Get(ctx, client.ObjectKey{
		Name:      applicationSecretName,
		Namespace: applicationSecretNamespace,
	}, &s)

	if errors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func deleteSecret(ctx context.Context, c client.Client) error {
	var s v1.Secret
	if err := c.Get(ctx, client.ObjectKey{
		Name:      applicationSecretName,
		Namespace: applicationSecretNamespace,
	}, &s); err != nil {
		return client.IgnoreNotFound(err)
	}

	if err := c.Delete(ctx, &s); err != nil {
		return err
	}
	return nil
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
