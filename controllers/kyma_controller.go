package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	eamapiv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	klmapiv1beta2 "github.com/kyma-project/lifecycle-manager/api/v1beta2"
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

	kyma := &klmapiv1beta2.Kyma{}
	err := r.Client.Get(ctx, req.NamespacedName, kyma)
	if err != nil {
		return kcontrollerruntime.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.createEventingAuth(ctx, kyma); err != nil {
		return kcontrollerruntime.Result{}, err
	}

	return kcontrollerruntime.Result{}, nil
}

func (r *KymaReconciler) createEventingAuth(ctx context.Context, kyma *klmapiv1beta2.Kyma) error {
	actual := &eamapiv1alpha1.EventingAuth{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: kyma.Namespace, Name: kyma.Name}, actual)
	if err != nil {
		if kapierrors.IsNotFound(err) {
			desired := &eamapiv1alpha1.EventingAuth{
				ObjectMeta: kmetav1.ObjectMeta{
					Namespace: kyma.Namespace,
					Name:      kyma.Name,
				},
			}
			if err = controllerutil.SetControllerReference(kyma, desired, r.Scheme); err != nil {
				return err
			}
			err = r.Client.Create(ctx, desired)
			if err != nil {
				return fmt.Errorf("failed to create EventingAuth resource: %w", err)
			}
			return nil
		}
		return errors.Wrap(err, "failed to retrieve EventingAuth resource")
	}

	desired := actual.DeepCopy()
	// remove previous controller ref regardless of currently specified api-version of the controller ref
	if controllerutil.HasControllerReference(desired) {
		ownerRefs := desired.GetOwnerReferences()
		controllerIndex := -1
		for i, ref := range ownerRefs {
			isTrue := ref.Controller
			if ref.Controller != nil && *isTrue {
				controllerIndex = i
				break
			}
		}
		if controllerIndex > -1 {
			ownerRefs = append(ownerRefs[:controllerIndex], ownerRefs[controllerIndex+1:]...)
		}
		desired.SetOwnerReferences(ownerRefs)
	}

	if err = controllerutil.SetControllerReference(kyma, desired, r.Scheme); err != nil {
		return err
	}
	if !reflect.DeepEqual(desired, actual) {
		err = r.Client.Update(ctx, desired, &client.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to patch EventingAuth resource")
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KymaReconciler) SetupWithManager(mgr kcontrollerruntime.Manager) error {
	return kcontrollerruntime.NewControllerManagedBy(mgr).
		For(&klmapiv1beta2.Kyma{}).
		Owns(&eamapiv1alpha1.EventingAuth{}).
		Complete(r)
}
