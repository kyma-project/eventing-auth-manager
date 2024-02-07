package controllers

import (
	"context"
	"time"

	eamapiv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	klmapiv1beta1 "github.com/kyma-project/lifecycle-manager/api/v1beta1"
	"github.com/pkg/errors"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kcontrollerruntime "sigs.k8s.io/controller-runtime"
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
func (r *KymaReconciler) Reconcile(ctx context.Context, req kcontrollerruntime.Request) (kcontrollerruntime.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Kyma resource")

	kyma := &klmapiv1beta1.Kyma{}
	err := r.Client.Get(ctx, req.NamespacedName, kyma)
	if err != nil {
		return kcontrollerruntime.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.createEventingAuth(ctx, kyma); err != nil {
		return kcontrollerruntime.Result{}, err
	}

	return kcontrollerruntime.Result{}, nil
}

func (r *KymaReconciler) createEventingAuth(ctx context.Context, kyma *klmapiv1beta1.Kyma) error {
	eventingAuth := &eamapiv1alpha1.EventingAuth{
		ObjectMeta: kmetav1.ObjectMeta{
			Namespace: kyma.Namespace,
			Name:      kyma.Name,
		},
	}

	err := r.Client.Get(ctx, types.NamespacedName{Namespace: eventingAuth.Namespace, Name: eventingAuth.Name}, eventingAuth)
	if err != nil {
		if kapierrors.IsNotFound(err) {
			if err = controllerutil.SetControllerReference(kyma, eventingAuth, r.Scheme); err != nil {
				return err
			}
			err = r.Client.Create(ctx, eventingAuth)
			if err != nil {
				return errors.Wrap(err, "failed to create EventingAuth resource")
			}
			return nil
		}
		return errors.Wrap(err, "failed to retrieve EventingAuth resource")
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KymaReconciler) SetupWithManager(mgr kcontrollerruntime.Manager) error {
	return kcontrollerruntime.NewControllerManagedBy(mgr).
		For(&klmapiv1beta1.Kyma{}).
		Owns(&eamapiv1alpha1.EventingAuth{}).
		Complete(r)
}
