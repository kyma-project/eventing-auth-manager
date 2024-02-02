package ias

import (
	"context"
	"fmt"

	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kpkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	urlString      = "url"
	usernameString = "username"
	passwordString = "password"
)

func NewCredentials(url, username, password string) *Credentials {
	return &Credentials{
		URL:      url,
		Username: username,
		Password: password,
	}
}

type Credentials struct {
	URL      string
	Username string
	Password string
}

// ReadCredentials fetches ias credentials from secret in the cluster. Reads from env vars if secret is missing.
var ReadCredentials = func(namespace, name string, k8sClient kpkgclient.Client) (*Credentials, error) {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	iasSecret := &kcorev1.Secret{}
	err := k8sClient.Get(context.TODO(), namespacedName, iasSecret)
	if err != nil {
		return nil, err
	}

	var exists bool
	var url, username, password []byte
	var errors error
	if url, exists = iasSecret.Data[urlString]; !exists {
		errors = fmt.Errorf("key %s is not found in ias secret", urlString)
	}
	if username, exists = iasSecret.Data[usernameString]; !exists {
		if errors != nil {
			errors = fmt.Errorf("%w: key %s is not found in ias secret", errors, usernameString)
		} else {
			errors = fmt.Errorf("key %s is not found in ias secret", usernameString)
		}
	}
	if password, exists = iasSecret.Data[passwordString]; !exists {
		if errors != nil {
			errors = fmt.Errorf("%w: key %s is not found in ias secret", errors, passwordString)
		} else {
			errors = fmt.Errorf("key %s is not found in ias secret", passwordString)
		}
	}
	if errors != nil {
		return nil, errors
	}
	iasConfig := NewCredentials(string(url), string(username), string(password))
	return iasConfig, nil
}
