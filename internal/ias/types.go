package ias

import (
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Application struct {
	id           string
	clientID     string
	clientSecret string
	tokenURL     string
	certsURL     string
}

func NewApplication(id, clientID, clientSecret, tokenURL, certsURL string) Application {
	return Application{
		id:           id,
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		certsURL:     certsURL,
	}
}

func (a Application) ToSecret(name, ns string) kcorev1.Secret {
	return kcorev1.Secret{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"client_id":     []byte(a.clientID),
			"client_secret": []byte(a.clientSecret),
			"token_url":     []byte(a.tokenURL),
			"certs_url":     []byte(a.certsURL),
		},
	}
}

func (a Application) GetID() string {
	return a.id
}
