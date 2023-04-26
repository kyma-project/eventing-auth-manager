package ias

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	ctlrClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IasUrlEnvVarName      string = "IAS_URL"
	IasUsernameEnvVarName string = "IAS_USERNAME"
	IasPasswordEnvVarName string = "IAS_PASSWORD"
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
func ReadCredentials(namespace, name string, k8sClient ctlrClient.Client) (*Credentials, error) {
	iasConfig := &Credentials{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	iasSecret := &corev1.Secret{}
	err := k8sClient.Get(context.Background(), namespacedName, iasSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			credentials, err := readFromEnvVars()
			if err != nil {
				return nil, err
			}
			return credentials, nil
		}
		return nil, fmt.Errorf("failed fetching the iasSecret: %s", namespacedName.String())
	}

	var exists bool
	var url []byte
	if url, exists = iasSecret.Data["url"]; !exists {
		return nil, fmt.Errorf("key 'url' not found in iasSecret " + namespacedName.String())
	}
	var username []byte
	if username, exists = iasSecret.Data["username"]; !exists {
		return nil, fmt.Errorf("key 'username' not found in iasSecret " + namespacedName.String())
	}
	var password []byte
	if password, exists = iasSecret.Data["password"]; !exists {
		return nil, fmt.Errorf("key 'password' not found in iasSecret: %s" + namespacedName.String())
	}
	iasConfig = NewCredentials(string(url[:]), string(username[:]), string(password[:]))
	return iasConfig, nil
}

func readFromEnvVars() (*Credentials, error) {
	// TODO: log like reading ias credentials from environment variables
	url := os.Getenv(IasUrlEnvVarName)
	var iasUrlErr, iasUserErr, iasPassErr error
	if len(url) == 0 {
		iasUrlErr = fmt.Errorf("%s is not set", IasUrlEnvVarName)
	}
	username := os.Getenv(IasUsernameEnvVarName)
	if len(url) == 0 {
		iasUserErr = fmt.Errorf("%s is not set", IasUsernameEnvVarName)
	}
	password := os.Getenv(IasPasswordEnvVarName)
	if len(url) == 0 {
		iasPassErr = fmt.Errorf("%s is not set", IasPasswordEnvVarName)
	}
	err := errors.Join(iasUrlErr, iasUserErr, iasPassErr)
	if err != nil {
		return nil, err
	}
	return NewCredentials(url, username, password), nil
}
