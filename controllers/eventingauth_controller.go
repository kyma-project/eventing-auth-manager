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
	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	operatorv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
)

const (
	applicationSecretName               = "eventing-webhook-auth"
	applicationSecretNamespace          = "kyma-system"
	eventingAuthFinalizerName           = "eventingauth.operator.kyma-project.io/finalizer"
	IasCredsSecretNamespace      string = "IAS_CREDS_SECRET_NAMESPACE"
	IasCredsSecretName           string = "IAS_CREDS_SECRET_NAME"
	defaultIasCredsNamespaceName string = "kcp-system"
	defaultIasCredsSecretName    string = "eventing-auth-ias-creds"
)

// eventingAuthReconciler reconciles a EventingAuth object
type eventingAuthReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	errorRequeuePeriod   time.Duration
	defaultRequeuePeriod time.Duration
	iasClient            ias.Client
	// existingIasApplications stores existing IAS apps in memory not to recreate again if exists
	existingIasApplications map[string]ias.Application
}

func NewEventingAuthReconciler(c client.Client, s *runtime.Scheme, errorRequeuePeriod time.Duration, defaultRequeuePeriod time.Duration) ManagedReconciler {
	return &eventingAuthReconciler{
		Client:                  c,
		Scheme:                  s,
		errorRequeuePeriod:      errorRequeuePeriod,
		defaultRequeuePeriod:    defaultRequeuePeriod,
		existingIasApplications: map[string]ias.Application{},
	}
}

// TODO: Check if conditions are correctly represented
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=watch,list
func (r *eventingAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling EventingAuth")

	cr, err := fetchEventingAuth(ctx, r.Client, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	targetK8sClient, err := getTargetClusterClient(r.Client, cr.Name)
	if err != nil {
		logger.Error(err, "Failed to retrieve client of target cluster")
		return ctrl.Result{
			RequeueAfter: r.errorRequeuePeriod,
		}, err
	}

	// sync IAS client credentials
	r.iasClient, err = r.getIasClient()
	if err != nil {
		return ctrl.Result{
			RequeueAfter: r.errorRequeuePeriod,
		}, err
	}
	// check DeletionTimestamp to determine if object is under deletion
	if cr.ObjectMeta.DeletionTimestamp.IsZero() {
		if err = r.addFinalizer(ctx, &cr); err != nil {
			return ctrl.Result{
				RequeueAfter: r.errorRequeuePeriod,
			}, err
		}
	} else {
		logger.Info("Handling deletion")
		if err = r.handleDeletion(ctx, r.iasClient, targetK8sClient, &cr); err != nil {
			return ctrl.Result{
				RequeueAfter: r.errorRequeuePeriod,
			}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	appSecretExists, err := hasTargetClusterApplicationSecret(ctx, targetK8sClient)
	if err != nil {
		logger.Error(err, "Failed to retrieve secret state from target cluster")
		return ctrl.Result{
			RequeueAfter: r.errorRequeuePeriod,
		}, err
	}

	// TODO: check if secret creation condition is also true, otherwise it never updates false secret ready condition
	if !appSecretExists {
		logger.Info("Creating IAS application")
		var createAppErr error
		iasApplication, appExists := r.existingIasApplications[cr.Name]
		if !appExists {
			iasApplication, createAppErr = r.iasClient.CreateApplication(ctx, cr.Name)
			if createAppErr != nil {
				logger.Error(createAppErr, "Failed to create IAS application")
				if err := r.updateEventingAuthStatus(ctx, &cr, operatorv1alpha1.ConditionApplicationReady, createAppErr); err != nil {
					return ctrl.Result{
						RequeueAfter: r.errorRequeuePeriod,
					}, err
				}
				return ctrl.Result{
					RequeueAfter: r.errorRequeuePeriod,
				}, createAppErr
			}
			r.existingIasApplications[cr.Name] = iasApplication
		}
		cr.Status.Application = &operatorv1alpha1.IASApplication{
			Name: cr.Name,
			UUID: iasApplication.GetId(),
		}
		if err := r.updateEventingAuthStatus(ctx, &cr, operatorv1alpha1.ConditionApplicationReady, nil); err != nil {
			return ctrl.Result{
				RequeueAfter: r.errorRequeuePeriod,
			}, err
		}

		// TODO: keep the IAS secret in memory to check for existence, otherwise, new IAS secret is recreated multiple times for error in case K8s secret creation fails.
		appSecret := iasApplication.ToSecret(applicationSecretName, applicationSecretNamespace)

		createSecretErr := targetK8sClient.Create(ctx, &appSecret)
		if createSecretErr != nil {
			logger.Error(createSecretErr, "Failed to create application secret")
			if err := r.updateEventingAuthStatus(ctx, &cr, operatorv1alpha1.ConditionSecretReady, createSecretErr); err != nil {
				return ctrl.Result{
					RequeueAfter: r.errorRequeuePeriod,
				}, err
			}
			return ctrl.Result{
				RequeueAfter: r.errorRequeuePeriod,
			}, createAppErr
		}
		logger.Info("Created IAS application secret")

		cr.Status.AuthSecret = &operatorv1alpha1.AuthSecret{
			ClusterId:      cr.Name,
			NamespacedName: fmt.Sprintf("%s/%s", appSecret.Namespace, appSecret.Name),
		}
		if err := r.updateEventingAuthStatus(ctx, &cr, operatorv1alpha1.ConditionSecretReady, nil); err != nil {
			return ctrl.Result{
				RequeueAfter: r.errorRequeuePeriod,
			}, err
		}
	}

	logger.Info("Reconciliation done")
	return ctrl.Result{
		RequeueAfter: r.defaultRequeuePeriod,
	}, nil
}

func (r *eventingAuthReconciler) getIasClient() (ias.Client, error) {
	namespace, name := getIasSecretNamespaceAndNameConfigs()
	newIasCredentials, err := ias.ReadCredentials(namespace, name, r.Client)
	if err != nil {
		return nil, err
	}
	// return from cache unless credentials are changed
	if r.iasClient != nil && reflect.DeepEqual(r.iasClient.GetCredentials(), newIasCredentials) {
		return r.iasClient, nil
	}
	// update IAS client if credentials are changed
	iasClient, err := ias.NewIasClient(newIasCredentials.URL, newIasCredentials.Username, newIasCredentials.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to createa a new IAS client: %v", err)
	}
	return iasClient, nil
}

func getIasSecretNamespaceAndNameConfigs() (string, string) {
	namespace := os.Getenv(IasCredsSecretNamespace)
	if len(namespace) == 0 {
		namespace = defaultIasCredsNamespaceName
	}
	name := os.Getenv(IasCredsSecretName)
	if len(name) == 0 {
		name = defaultIasCredsSecretName
	}
	return namespace, name
}

// Adds the finalizer if none exists
func (r *eventingAuthReconciler) addFinalizer(ctx context.Context, cr *operatorv1alpha1.EventingAuth) error {
	if !controllerutil.ContainsFinalizer(cr, eventingAuthFinalizerName) {
		ctrl.Log.Info("Adding finalizer", "eventingAuth", cr.Name, "eventingAuthNamespace", cr.Namespace)
		controllerutil.AddFinalizer(cr, eventingAuthFinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return fmt.Errorf("failed to add finalizer: %v", err)
		}
	}
	return nil
}

// Deletes the secret and IAS app. Finally, removes the finalizer.
func (r *eventingAuthReconciler) handleDeletion(ctx context.Context, iasClient ias.Client, targetClusterClient client.Client, cr *operatorv1alpha1.EventingAuth) error {
	// The object is being deleted
	if controllerutil.ContainsFinalizer(cr, eventingAuthFinalizerName) {
		// delete k8s secret
		if err := deleteSecret(ctx, targetClusterClient); err != nil {
			return fmt.Errorf("failed to delete secret: %v", err)
		}
		ctrl.Log.Info("Deleted IAS application secret on target cluster", "eventingAuth", cr.Name, "namespace", cr.Namespace)

		// delete IAS application clean-up
		if err := iasClient.DeleteApplication(ctx, cr.Name); err != nil {
			return fmt.Errorf("failed to delete IAS Application: %v", err)
		}
		// delete the app from the cache
		delete(r.existingIasApplications, cr.Name)
		ctrl.Log.Info("Deleted IAS Application", "name", cr.Name)

		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(cr, eventingAuthFinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return fmt.Errorf("failed to remove finalizer: %v", err)
		}
	}
	return nil
}

// updateEventingAuthStatus updates the subscription's status changes to k8s.
func (r *eventingAuthReconciler) updateEventingAuthStatus(ctx context.Context, cr *operatorv1alpha1.EventingAuth, conditionType operatorv1alpha1.ConditionType, errToCheck error) error {
	_, err := operatorv1alpha1.UpdateConditionAndState(cr, conditionType, errToCheck)
	if err != nil {
		return err
	}

	namespacedName := &types.NamespacedName{
		Name:      cr.Name,
		Namespace: cr.Namespace,
	}

	// fetch the latest EventingAuth object, to avoid k8s conflict errors
	actualEventingAuth := &operatorv1alpha1.EventingAuth{}
	if err := r.Client.Get(ctx, *namespacedName, actualEventingAuth); err != nil {
		return err
	}

	// copy new changes to the latest object
	desiredEventingAuth := actualEventingAuth.DeepCopy()
	desiredEventingAuth.Status = cr.Status

	// sync EventingAuth status with k8s
	if err = r.updateStatus(ctx, actualEventingAuth, desiredEventingAuth); err != nil {
		return fmt.Errorf("failed to update EventingAuth status: %v", err)
	}

	return nil
}

func (r *eventingAuthReconciler) updateStatus(ctx context.Context, oldEventingAuth, newEventingAuth *operatorv1alpha1.EventingAuth) error {
	// compare the status taking into consideration lastTransitionTime in conditions
	if operatorv1alpha1.IsEventingAuthStatusEqual(oldEventingAuth.Status, newEventingAuth.Status) {
		return nil
	}
	return r.Client.Status().Update(ctx, newEventingAuth)
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

	if apiErrors.IsNotFound(err) {
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
