package ias

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Application struct {
	id           string
	clientId     string
	clientSecret string
	tokenUrl     string
	certsUrl     string
}

func NewApplication(id, clientId, clientSecret, tokenUrl, certsUrl string) Application {
	return Application{
		id:           id,
		clientId:     clientId,
		clientSecret: clientSecret,
		tokenUrl:     tokenUrl,
		certsUrl:     certsUrl,
	}
}

func (a Application) ToSecret(name, ns string) v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"client_id":     []byte(a.clientId),
			"client_secret": []byte(a.clientSecret),
			"token_url":     []byte(a.tokenUrl),
			"certs_url":     []byte(a.certsUrl),
		},
	}
}

func (a Application) GetId() string {
	return a.id
}
