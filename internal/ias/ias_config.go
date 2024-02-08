package ias

import (
	"context"

	"github.com/pkg/errors"
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
var ReadCredentials = func(namespace, name string, k8sClient kpkgclient.Client) (*Credentials, error) { //nolint:gochecknoglobals // For mocking purposes.
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	iasSecret := &kcorev1.Secret{}
	if err := k8sClient.Get(context.TODO(), namespacedName, iasSecret); err != nil {
		return nil, err
	}

	var exists bool
	var url, username, password []byte
	var err error
	if url, exists = iasSecret.Data[urlString]; !exists {
		err = errors.Errorf("key %s is not found in ias secret", urlString)
	}
	if username, exists = iasSecret.Data[usernameString]; !exists {
		if err != nil {
			err = errors.Wrapf(err, "key %s is not found in ias secret", usernameString)
		} else {
			err = errors.Errorf("key %s is not found in ias secret", usernameString)
		}
	}
	if password, exists = iasSecret.Data[passwordString]; !exists {
		if err != nil {
			err = errors.Wrapf(err, "key %s is not found in ias secret", passwordString)
		} else {
			err = errors.Errorf("key %s is not found in ias secret", passwordString)
		}
	}
	if err != nil {
		return nil, err
	}
	iasConfig := NewCredentials(string(url), string(username), string(password))
	return iasConfig, nil
}
