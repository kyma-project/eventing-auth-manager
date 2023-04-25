package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getTargetClusterClient(k8sClient client.Client, targetClusterId string) (client.Client, error) {
	kubeconfigSecretName := fmt.Sprintf("kubeconfig-%s", targetClusterId)

	secret := &v1.Secret{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: kubeconfigSecretName, Namespace: "kcp-system"}, secret); err != nil {
		return nil, err
	}

	encoded := secret.Data["config"]
	if len(encoded) == 0 {
		return nil, fmt.Errorf("failed to find target cluster kubeconfig in secret %s", kubeconfigSecretName)
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	n, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		return nil, err
	}
	kubeconfig := decoded[:n]
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{})
}
