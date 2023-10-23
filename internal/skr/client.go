package skr

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplicationSecretName      = "eventing-webhook-auth"
	ApplicationSecretNamespace = "kyma-system"
	KcpNamespace               = "kcp-system"
)

type Client interface {
	DeleteSecret(ctx context.Context) error
	HasApplicationSecret(ctx context.Context) (bool, error)
	CreateSecret(ctx context.Context, app ias.Application) (v1.Secret, error)
}

type client struct {
	k8sClient ctrlclient.Client
}

var NewClient = func(k8sClient ctrlclient.Client, skrClusterId string) (Client, error) {
	kubeconfigSecretName := fmt.Sprintf("kubeconfig-%s", skrClusterId)

	secret := &v1.Secret{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: kubeconfigSecretName, Namespace: KcpNamespace}, secret); err != nil {
		return nil, err
	}

	kubeconfig := secret.Data["config"]
	if len(kubeconfig) == 0 {
		return nil, fmt.Errorf("failed to find SKR cluster kubeconfig in secret %s", kubeconfigSecretName)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	c, err := ctrlclient.New(config, ctrlclient.Options{})
	if err != nil {
		return nil, err
	}

	return &client{k8sClient: c}, nil
}

func (c *client) DeleteSecret(ctx context.Context) error {
	var s v1.Secret
	if err := c.k8sClient.Get(ctx, ctrlclient.ObjectKey{
		Name:      ApplicationSecretName,
		Namespace: ApplicationSecretNamespace,
	}, &s); err != nil {
		return ctrlclient.IgnoreNotFound(err)
	}

	if err := c.k8sClient.Delete(ctx, &s); err != nil {
		return err
	}
	return nil
}

func (c *client) CreateSecret(ctx context.Context, app ias.Application) (v1.Secret, error) {
	appSecret := app.ToSecret(ApplicationSecretName, ApplicationSecretNamespace)
	err := c.k8sClient.Create(ctx, &appSecret)
	return appSecret, err
}

func (c *client) HasApplicationSecret(ctx context.Context) (bool, error) {
	var s v1.Secret
	err := c.k8sClient.Get(ctx, ctrlclient.ObjectKey{
		Name:      ApplicationSecretName,
		Namespace: ApplicationSecretNamespace,
	}, &s)

	if apiErrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
