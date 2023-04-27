package ias

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctlrClient "sigs.k8s.io/controller-runtime/pkg/client"
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
var ReadCredentials = func(namespace, name string, k8sClient ctlrClient.Client) (*Credentials, error) {
	iasConfig := &Credentials{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	iasSecret := &corev1.Secret{}
	err := k8sClient.Get(context.TODO(), namespacedName, iasSecret)
	if err != nil {
		return nil, err
	}

	var exists bool
	var url, username, password []byte
	var errUrl, errUsername, errPassword error
	if url, exists = iasSecret.Data[urlString]; !exists {
		errUrl = fmt.Errorf("key %s is not found in ias secret", urlString)
	}
	if username, exists = iasSecret.Data[usernameString]; !exists {
		errUsername = fmt.Errorf("key %s is not found in ias secret", usernameString)
	}
	if password, exists = iasSecret.Data[passwordString]; !exists {
		errPassword = fmt.Errorf("key %s is not found in ias secret", passwordString)
	}
	err = errors.Join(errUrl, errUsername, errPassword)
	if err != nil {
		return nil, err
	}
	iasConfig = NewCredentials(string(url[:]), string(username[:]), string(password[:]))
	return iasConfig, nil
}
