package controllers_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	"github.com/kyma-project/eventing-auth-manager/internal/skr"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
)

var (
	originalNewIasClientFunc    func(iasTenantUrl, user, password string) (ias.Client, error)
	originalReadCredentialsFunc func(namespace, name string, k8sClient client.Client) (*ias.Credentials, error)
	originalNewSkrClientFunc    func(k8sClient client.Client, targetClusterId string) (skr.Client, error)
)

func stubSuccessfulIasAppCreation() {
	By("Stubbing IAS application creation to succeed")
	stubIasAppCreation(iasClientStub{})
}

func stubFailedIasAppCreation() {
	By("Stubbing IAS application creation to fail")
	stubIasAppCreation(appCreationFailsIasClientStub{})
}

func stubIasAppCreation(c ias.Client) {
	// The IAS client is initialized once in the "Reconcile" method of the controller. To update the IAS client stub by forcing a replacement, we need to
	// update the IAS credentials stub so that the implemented logic assumes that the IAS credentials have been rotated and forces a reinitialization of the IAS client.
	replaceIasReadCredentialsWithStub(ias.Credentials{URL: iasUrl, Username: uuid.New().String(), Password: iasPassword})
	replaceIasNewIasClientWithStub(c)
}

type iasClientStub struct {
}

func (i iasClientStub) CreateApplication(_ context.Context, name string) (ias.Application, error) {
	return ias.NewApplication(
		fmt.Sprintf("id-for-%s", name),
		fmt.Sprintf("client-id-for-%s", name),
		"test-client-secret",
		"https://test-token-url.com/token",
		"https://test-token-url.com/certs",
	), nil
}

func (i iasClientStub) DeleteApplication(_ context.Context, _ string) error {
	return nil
}

func (i iasClientStub) GetCredentials() *ias.Credentials {
	return &ias.Credentials{}
}

type appCreationFailsIasClientStub struct {
	iasClientStub
}

func (i appCreationFailsIasClientStub) CreateApplication(_ context.Context, _ string) (ias.Application, error) {
	return ias.Application{}, errors.New("stubbed IAS application creation error")
}

func replaceIasReadCredentialsWithStub(credentials ias.Credentials) {
	ias.ReadCredentials = func(namespace, name string, k8sClient client.Client) (*ias.Credentials, error) {
		return &credentials, nil
	}
}

func storeOriginalsOfStubbedFunctions() {
	originalReadCredentialsFunc = ias.ReadCredentials
	originalNewIasClientFunc = ias.NewClient
	originalNewSkrClientFunc = skr.NewClient
}
func revertReadCredentialsStub() {
	ias.ReadCredentials = originalReadCredentialsFunc
}
func revertIasNewClientStub() {
	ias.NewClient = originalNewIasClientFunc
}
func revertSkrNewClientStub() {
	skr.NewClient = originalNewSkrClientFunc
}

func replaceIasNewIasClientWithStub(c ias.Client) {
	ias.NewClient = func(iasTenantUrl, user, password string) (ias.Client, error) {
		return c, nil
	}
}

type skrClientStub struct {
	skr.Client
}

func (s skrClientStub) CreateSecret(_ context.Context, app ias.Application) (v1.Secret, error) {
	return app.ToSecret(skr.ApplicationSecretName, skr.ApplicationSecretNamespace), nil
}
func (s skrClientStub) HasApplicationSecret(_ context.Context) (bool, error) {
	return false, nil
}

func (s skrClientStub) DeleteSecret(_ context.Context) error {
	return nil
}

func stubSuccessfulSkrSecretCreation() {
	By("Stubbing SKR secret creation to succeed")
	replaceSkrClientWithStub(skrClientStub{})
}

func stubFailedSkrSecretCreation() {
	By("Stubbing SKR secret creation to fail")
	replaceSkrClientWithStub(secretCreationFailedSkrClientStub{})
}

type secretCreationFailedSkrClientStub struct {
	skrClientStub
}

func (s secretCreationFailedSkrClientStub) CreateSecret(_ context.Context, _ ias.Application) (v1.Secret, error) {
	return v1.Secret{}, errors.New("stubbed skr secret creation error")
}

func replaceSkrClientWithStub(c skr.Client) {
	skr.NewClient = func(k8sClient client.Client, targetClusterId string) (skr.Client, error) {
		return c, nil
	}
}
