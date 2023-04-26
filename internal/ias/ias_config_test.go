package ias

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/mocks"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctlrClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_ReadCredentials(t *testing.T) {
	testNamespace := "mock-namespace"
	testName := "mock-name"
	testUrl := "https://test.url.com"
	testUsername := "test-username"
	testPassword := "test-password"
	mockSecret := createMockSecret(testNamespace, testName, testUrl, testUsername, testPassword)

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
				MockSecret: mockSecret,
			},
			wantCredentials: Credentials{
				URL:      testUrl,
				Username: testUsername,
				Password: testPassword,
			},
		},
		{
			name: "Fails with an unexpected error",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return errors.New("mock unexpected error")
				},
				MockSecret: mockSecret,
			},
			wantError: fmt.Errorf("failed fetching the iasSecret: %s", testNamespace+"/"+testName),
		},
		{
			name: "Read from environment variables if secret is missing",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return &apiErrors.StatusError{
						ErrStatus: metav1.Status{
							Reason: "resource not found",
							Code:   404,
						},
					}
				},
				MockSecret: mockSecret,
			},
			mockEnvVars: func() {
				os.Setenv(IasUrlEnvVarName, testUrl)
				os.Setenv(IasUsernameEnvVarName, testUsername)
				os.Setenv(IasPasswordEnvVarName, testPassword)
			},
			wantCredentials: Credentials{
				URL:      testUrl,
				Username: testUsername,
				Password: testPassword,
			},
		},
		{
			name: "Fails if both secret and environment vars are missing",
			givenK8sClientMock: &mocks.MockClient{
				MockFunction: func() error {
					return &apiErrors.StatusError{
						ErrStatus: metav1.Status{
							Reason: "resource not found",
							Code:   404,
						},
					}
				},
				MockSecret: mockSecret,
			},
			mockEnvVars: func() {
				os.Clearenv()
			},
			wantError: fmt.Errorf("%s is not set\n%s is not set\n%s is not set", IasUrlEnvVarName, IasUsernameEnvVarName, IasPasswordEnvVarName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			if tt.mockEnvVars != nil {
				tt.mockEnvVars()
			}
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
	s.Data["url"] = []byte(testUrl)
	s.Data["username"] = []byte(testUsername)
	s.Data["password"] = []byte(testPassword)
	return s
}
