package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	kymav1beta1 "github.com/kyma-project/lifecycle-manager/api/v1beta1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KymaReconciler reconciles a Kyma resource.
type KymaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	time.Duration
}

func NewKymaReconciler(c client.Client, s *runtime.Scheme) *KymaReconciler {
	return &KymaReconciler{
		Client: c,
		Scheme: s,
	}
}

// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=kymas,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventingauths/status,verbs=get;list
func (r *KymaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Kyma resource")

	kyma := &kymav1beta1.Kyma{}
	err := r.Client.Get(ctx, req.NamespacedName, kyma)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.createEventingAuth(ctx, kyma); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KymaReconciler) createEventingAuth(ctx context.Context, kyma *kymav1beta1.Kyma) error {
	eventingAuth := &v1alpha1.EventingAuth{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kyma.Namespace,
			Name:      kyma.Name,
		},
	}

	err := r.Client.Get(ctx, types.NamespacedName{Namespace: eventingAuth.Namespace, Name: eventingAuth.Name}, eventingAuth)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if err = controllerutil.SetControllerReference(kyma, eventingAuth, r.Scheme); err != nil {
				return err
			}
			err = r.Client.Create(ctx, eventingAuth)
			if err != nil {
				return fmt.Errorf("failed to create EventingAuth resource: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to retrieve EventingAuth resource: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KymaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kymav1beta1.Kyma{}).
		Owns(&v1alpha1.EventingAuth{}).
		Complete(r)
}
