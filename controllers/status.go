package controllers

import (
	"context"

	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func updateStatus(ctx context.Context, client client.Client, cr v1alpha1.EventingAuth, state v1alpha1.State) error {
	cr.Status.State = state
	return client.Status().Update(ctx, &cr)
}
