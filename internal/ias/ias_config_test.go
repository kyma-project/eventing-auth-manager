package ias

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/mocks"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctlrClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_ReadCredentials(t *testing.T) {
	testNamespace := "mock-namespace"
	testName := "mock-name"
	testUrl := "https://test.url.com"
	testUsername := "test-username"
	testPassword := "test-password"

	tests := []struct {
		name               string
		givenK8sClientMock ctlrClient.Client
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
			wantError: fmt.Errorf("key %s is not found in ias secret\nkey %s is not found in ias secret\nkey %s is not found in ias secret",
				urlString, usernameString, passwordString),
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

func createMockSecret(testNamespace, testName, testUrl, testUsername, testPassword string) *v1.Secret {
	s := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
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

func createMockSecretWithoutDataFields(testNamespace, testName string) *v1.Secret {
	s := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testNamespace,
			Namespace: testName,
		},
		Data: map[string][]byte{},
	}
	s.Data = map[string][]byte{}
	return s
}
