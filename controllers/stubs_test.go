package controllers_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	eamias "github.com/kyma-project/eventing-auth-manager/internal/ias"
	"github.com/kyma-project/eventing-auth-manager/internal/skr"
	kcorev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
)

var (
	originalNewIasClientFunc    func(iasTenantUrl, user, password string) (eamias.Client, error)
	originalReadCredentialsFunc func(namespace, name string, k8sClient client.Client) (*eamias.Credentials, error)
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

func stubIasAppCreation(c eamias.Client) {
	// The IAS client is initialized once in the "Reconcile" method of the controller. To update the IAS client stub by forcing a replacement, we need to
	// update the IAS credentials stub so that the implemented logic assumes that the IAS credentials have been rotated and forces a reinitialization of the IAS client.
	replaceIasReadCredentialsWithStub(eamias.Credentials{URL: iasUrl, Username: uuid.New().String(), Password: iasPassword})
	replaceIasNewIasClientWithStub(c)
}

type iasClientStub struct {
}

func (i iasClientStub) CreateApplication(_ context.Context, name string) (eamias.Application, error) {
	return eamias.NewApplication(
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

func (i iasClientStub) GetCredentials() *eamias.Credentials {
	return &eamias.Credentials{}
}

type appCreationFailsIasClientStub struct {
	iasClientStub
}

func (i appCreationFailsIasClientStub) CreateApplication(_ context.Context, _ string) (eamias.Application, error) {
	return eamias.Application{}, errors.New("stubbed IAS application creation error")
}

func replaceIasReadCredentialsWithStub(credentials eamias.Credentials) {
	eamias.ReadCredentials = func(namespace, name string, k8sClient client.Client) (*eamias.Credentials, error) {
		return &credentials, nil
	}
}

func storeOriginalsOfStubbedFunctions() {
	originalReadCredentialsFunc = eamias.ReadCredentials
	originalNewIasClientFunc = eamias.NewClient
	originalNewSkrClientFunc = skr.NewClient
}
func revertReadCredentialsStub() {
	eamias.ReadCredentials = originalReadCredentialsFunc
}
func revertIasNewClientStub() {
	eamias.NewClient = originalNewIasClientFunc
}
func revertSkrNewClientStub() {
	skr.NewClient = originalNewSkrClientFunc
}

func replaceIasNewIasClientWithStub(c eamias.Client) {
	eamias.NewClient = func(iasTenantUrl, user, password string) (eamias.Client, error) {
		return c, nil
	}
}

type skrClientStub struct {
	skr.Client
}

func (s skrClientStub) CreateSecret(_ context.Context, app eamias.Application) (kcorev1.Secret, error) {
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

func (s secretCreationFailedSkrClientStub) CreateSecret(_ context.Context, _ eamias.Application) (kcorev1.Secret, error) {
	return kcorev1.Secret{}, errors.New("stubbed skr secret creation error")
}

func replaceSkrClientWithStub(c skr.Client) {
	skr.NewClient = func(k8sClient client.Client, targetClusterId string) (skr.Client, error) {
		return c, nil
	}
}
