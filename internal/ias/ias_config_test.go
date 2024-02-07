package ias

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/mocks"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kpkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_ReadCredentials(t *testing.T) {
	testNamespace := "mock-namespace"
	testName := "mock-name"
	testUrl := "https://test.url.com"
	testUsername := "test-username"
	testPassword := "test-password"

	tests := []struct {
		name               string
		givenK8sClientMock kpkgclient.Client
		mockEnvVars        func()
		wantCredentials    Credentials
		wantError          error
	}{
		{
			name: "Reads credentials from secret successfully",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return nil
				},
				MockSecret: createMockSecret(testNamespace, testName, testUrl, testUsername, testPassword),
			},
			wantCredentials: Credentials{
				URL:      testUrl,
				Username: testUsername,
				Password: testPassword,
			},
		},
		{
			name: "Fails with an error",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return errors.New("mock error")
				},
				MockSecret: createMockSecret(testNamespace, testName, testUrl, testUsername, testPassword),
			},
			wantError: errors.New("mock error"),
		},
		{
			name: "Fails with missing data fields error",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return nil
				},
				MockSecret: createMockSecretWithoutDataFields(testNamespace, testPassword),
			},
			wantError: fmt.Errorf("key %s is not found in ias secret: key %s is not found in ias secret: key %s is not found in ias secret",
				urlString, usernameString, passwordString),
		},
		{
			name: "Fails with missing username data fields error",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return nil
				},
				MockSecret: createMockSecretWithoutUsernameDataFields(testNamespace, testName, testUrl, testPassword),
			},
			wantError: fmt.Errorf("key %s is not found in ias secret", usernameString),
		},
		{
			name: "Fails with missing username and password data fields error",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return nil
				},
				MockSecret: createMockSecretWithoutUsernameAndPasswordDataFields(testNamespace, testName, testUrl),
			},
			wantError: fmt.Errorf("key %s is not found in ias secret: key %s is not found in ias secret", usernameString, passwordString),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			actualCredentials, err := ReadCredentials(testNamespace, testName, tt.givenK8sClientMock)
			// then
			if tt.wantError != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantError, err.Error())
			} else {
				require.Equal(t, tt.wantCredentials, *actualCredentials)
			}
		})
	}
}

func createMockSecret(testNamespace, testName, testUrl, testUsername, testPassword string) *kcorev1.Secret {
	s := &kcorev1.Secret{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      testNamespace,
			Namespace: testName,
		},
		Data: map[string][]byte{},
	}
	s.Data[urlString] = []byte(testUrl)
	s.Data[usernameString] = []byte(testUsername)
	s.Data[passwordString] = []byte(testPassword)
	return s
}

func createMockSecretWithoutDataFields(testNamespace, testName string) *kcorev1.Secret {
	s := &kcorev1.Secret{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      testNamespace,
			Namespace: testName,
		},
		Data: map[string][]byte{},
	}
	s.Data = map[string][]byte{}
	return s
}

func createMockSecretWithoutUsernameDataFields(testNamespace, testName, testUrl, testPassword string) *kcorev1.Secret {
	s := &kcorev1.Secret{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      testNamespace,
			Namespace: testName,
		},
		Data: map[string][]byte{},
	}
	s.Data[urlString] = []byte(testUrl)
	s.Data[passwordString] = []byte(testPassword)
	return s
}

func createMockSecretWithoutUsernameAndPasswordDataFields(testNamespace, testName, testUrl string) *kcorev1.Secret {
	s := &kcorev1.Secret{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      testNamespace,
			Namespace: testName,
		},
		Data: map[string][]byte{},
	}
	s.Data[urlString] = []byte(testUrl)
	return s
}
