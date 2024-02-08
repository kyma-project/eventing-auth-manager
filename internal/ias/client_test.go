package ias

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/api"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/api/mocks"
	eamoidcmocks "github.com/kyma-project/eventing-auth-manager/internal/ias/internal/oidc/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func Test_CreateApplication(t *testing.T) {
	appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
	tests := []struct {
		name               string
		givenAPIMock       func() *mocks.ClientWithResponsesInterface
		oidcClientMock     *eamoidcmocks.Client
		clientTokenURLMock *string
		clientJWKSURIMock  *string
		assertCalls        func(*testing.T, *mocks.ClientWithResponsesInterface)
		wantApp            Application
		wantError          error
	}{
		{
			name: "should create new application when fetching existing applications returns status 200 and no applications",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, appID)
				mockGetApplicationWithResponseStatusOK(&clientMock, appID)

				return &clientMock
			},
			oidcClientMock: mockClient(
				t,
				ptr.To("https://test.com/token"),
				ptr.To("https://test.com/certs"),
			),
			wantApp: NewApplication(
				appID.String(),
				"clientIdMock",
				"clientSecretMock",
				"https://test.com/token",
				"https://test.com/certs",
			),
		},
		{
			name: "should create new application when fetching existing applications returns status 404",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")

				mockGetAllApplicationsWithResponseStatusNotFound(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, appID)
				mockGetApplicationWithResponseStatusOK(&clientMock, appID)

				return &clientMock
			},
			oidcClientMock: mockClient(
				t,
				ptr.To("https://test.com/token"),
				ptr.To("https://test.com/certs"),
			),
			wantApp: NewApplication(
				appID.String(),
				"clientIdMock",
				"clientSecretMock",
				"https://test.com/token",
				"https://test.com/certs",
			),
		},
		{
			name: "should recreate application when application already exists",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				existingAppID := uuid.MustParse("5ab797c0-80a0-4ca4-ad7f-50a0f40231d6")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, existingAppID)
				mockDeleteApplicationWithResponseStatusOk(&clientMock, existingAppID)

				newAppID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponseStatusCreated(&clientMock, newAppID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, newAppID)
				mockGetApplicationWithResponseStatusOK(&clientMock, newAppID)

				return &clientMock
			},
			oidcClientMock: mockClient(
				t,
				ptr.To("https://test.com/token"),
				ptr.To("https://test.com/certs"),
			),
			wantApp: NewApplication(
				appID.String(),
				"clientIdMock",
				"clientSecretMock",
				"https://test.com/token",
				"https://test.com/certs",
			),
		},
		{
			name: "should return an error when multiple applications exist for the given name",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appID2 := uuid.MustParse("41de6fec-e0fc-47d7-b35c-3b19c4927e4f")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appID, appID2)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("found multiple applications with the same name Test-App-Name"), //nolint:goerr113 // used one time only in tests.
		},
		{
			name: "should return error when application ID can't be retrieved from location header",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, "non-uuid-application-id")

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to retrieve application ID from header: invalid UUID length: 23"), //nolint:goerr113 // used one time only in tests.
		},
		{
			name: "should return error when fetching existing application failed",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				mockGetAllApplicationsWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errFetchExistingApplications,
		},
		{
			name: "should return error when application exists and deletion failed",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appID)
				mockDeleteApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errDeleteExistingApplicationBeforeCreation,
		},
		{
			name: "should return error when application is not created",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errCreateApplication,
		},
		{
			name: "should return error when secret is not created",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)

				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errCreateAPISecret,
		},
		{
			name: "should return error when client id wasn't fetched",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, appID)
				mockGetApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errRetrieveClientID,
		},
		{
			name: "should return an error when token URL wasn't fetched",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, appID)
				mockGetApplicationWithResponseStatusOK(&clientMock, appID)

				return &clientMock
			},
			oidcClientMock: mockGetTokenEndpoint(t, nil),
			wantApp:        Application{},
			wantError:      errFetchTokenURL,
		},
		{
			name: "should return an error when jwks URI wasn't fetched",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, appID)
				mockGetApplicationWithResponseStatusOK(&clientMock, appID)

				return &clientMock
			},
			oidcClientMock: mockClient(t, ptr.To("https://test.com/token"), nil),
			wantApp:        Application{},
			wantError:      errFetchJWKSURI,
		},
		{
			name: "should create new application without fetching token URL when it is already cached in the client",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appID.String())
				mockCreateAPISecretWithResponseStatusCreated(&clientMock, appID)
				mockGetApplicationWithResponseStatusOK(&clientMock, appID)

				return &clientMock
			},
			clientTokenURLMock: ptr.To("https://from-cache.com/token"),
			clientJWKSURIMock:  ptr.To("https://from-cache.com/certs"),
			wantApp: NewApplication(
				appID.String(),
				"clientIdMock",
				"clientSecretMock",
				"https://from-cache.com/token",
				"https://from-cache.com/certs",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiMock := tt.givenAPIMock()
			oidcMock := tt.oidcClientMock

			client := client{
				api:        apiMock,
				oidcClient: oidcMock,
				tokenURL:   tt.clientTokenURLMock,
				jwksURI:    tt.clientJWKSURIMock,
			}

			// when
			app, err := client.CreateApplication(context.TODO(), "Test-App-Name")

			// then
			require.Equal(t, tt.wantApp, app)

			if tt.wantError != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantError, err.Error())
			} else {
				require.NoError(t, err)
			}

			if tt.assertCalls != nil {
				tt.assertCalls(t, apiMock)
			} else {
				apiMock.AssertExpectations(t)
			}

			if oidcMock != nil {
				oidcMock.AssertExpectations(t)
			}
		})
	}
}

func Test_DeleteApplication(t *testing.T) {
	tests := []struct {
		name         string
		givenAPIMock func() *mocks.ClientWithResponsesInterface
		wantError    error
	}{
		{
			name: "should delete application",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appID)
				mockDeleteApplicationWithResponseStatusOk(&clientMock, appID)

				return &clientMock
			},
		},
		{
			name: "should return error when application is not deleted",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appID)
				mockDeleteApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantError: errDeleteApplication,
		},
		{
			name: "should not return an error when application doesn't exist",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)

				return &clientMock
			},
		},
		{
			name: "should not return an error when app was found, but was already deleted",
			givenAPIMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appID := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appID)
				mockDeleteApplicationWithResponseStatusNotFound(&clientMock)

				return &clientMock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiMock := tt.givenAPIMock()

			client := client{
				api: apiMock,
			}

			// when
			err := client.DeleteApplication(context.TODO(), "Test-App-Name")

			// then
			if tt.wantError != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantError, err.Error())
			} else {
				require.NoError(t, err)
			}

			apiMock.AssertExpectations(t)
		})
	}
}

func mockGetAllApplicationsWithResponseStatusInternalServerError(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("GetAllApplicationsWithResponse", mock.Anything, mock.Anything).
		Return(&api.GetAllApplicationsResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)
}

func mockGetAllApplicationsWithResponseStatusNotFound(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("GetAllApplicationsWithResponse", mock.Anything, mock.Anything).
		Return(&api.GetAllApplicationsResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil)
}

func mockGetAllApplicationsWithResponseStatusOkEmptyResponse(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("GetAllApplicationsWithResponse", mock.Anything, mock.Anything).
		Return(&api.GetAllApplicationsResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &api.ApplicationsResponse{
				Applications: &[]api.ApplicationResponse{},
			},
		}, nil)
}

func mockGetAllApplicationsWithResponseStatusOk(clientMock *mocks.ClientWithResponsesInterface, appIds ...uuid.UUID) {
	appResponses := make([]api.ApplicationResponse, 0)
	for _, appID := range appIds {
		id := appID
		appResponses = append(appResponses, api.ApplicationResponse{
			Id: &id,
		})
	}

	appsFilter := "name eq Test-App-Name"
	clientMock.On("GetAllApplicationsWithResponse", mock.Anything, &api.GetAllApplicationsParams{Filter: &appsFilter}).
		Return(&api.GetAllApplicationsResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &api.ApplicationsResponse{
				Applications: &appResponses,
			},
		}, nil)
}

func mockCreateApplicationWithResponseStatusInternalServerError(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("CreateApplicationWithResponse", mock.Anything, mock.Anything, newIasApplication("Test-App-Name")).
		Return(&api.CreateApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)
}

func mockCreateApplicationWithResponseStatusCreated(clientMock *mocks.ClientWithResponsesInterface, appID string) {
	clientMock.On("CreateApplicationWithResponse", mock.Anything, mock.Anything, newIasApplication("Test-App-Name")).
		Return(&api.CreateApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusCreated,
				Header: map[string][]string{
					"Location": {fmt.Sprintf("https://test.com/v1/Applications/%s", appID)},
				},
			},
		}, nil)
}

func mockCreateAPISecretWithResponseStatusInternalServerError(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("CreateApiSecretWithResponse", mock.Anything, mock.Anything, mock.Anything).
		Return(&api.CreateApiSecretResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)
}

func mockCreateAPISecretWithResponseStatusCreated(clientMock *mocks.ClientWithResponsesInterface, appID uuid.UUID) {
	d := "eventing-auth-manager"
	s := "clientSecretMock"
	secretRequest := api.CreateApiSecretJSONRequestBody{
		AuthorizationScopes: &[]api.AuthorizationScope{"oAuth"},
		Description:         &d,
	}
	clientMock.On("CreateApiSecretWithResponse", mock.Anything, appID, secretRequest).
		Return(&api.CreateApiSecretResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusCreated,
			},
			JSON201: &api.ApiSecretResponse{
				Secret: &s,
			},
		}, nil)
}

func mockGetApplicationWithResponseStatusInternalServerError(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("GetApplicationWithResponse", mock.Anything, mock.Anything, mock.Anything).
		Return(&api.GetApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)
}

func mockGetApplicationWithResponseStatusOK(clientMock *mocks.ClientWithResponsesInterface, appID uuid.UUID) {
	cID := "clientIdMock"
	clientMock.On("GetApplicationWithResponse", mock.Anything, appID, mock.Anything).
		Return(&api.GetApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &api.ApplicationResponse{
				UrnSapIdentityApplicationSchemasExtensionSci10Authentication: &api.AuthenticationSchema{
					ClientId: &cID,
				},
			},
		}, nil)
}

func mockDeleteApplicationWithResponseStatusOk(clientMock *mocks.ClientWithResponsesInterface, appID uuid.UUID) {
	clientMock.On("DeleteApplicationWithResponse", mock.Anything, appID).
		Return(&api.DeleteApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil)
}

func mockDeleteApplicationWithResponseStatusInternalServerError(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("DeleteApplicationWithResponse", mock.Anything, mock.Anything).
		Return(&api.DeleteApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)
}

func mockDeleteApplicationWithResponseStatusNotFound(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("DeleteApplicationWithResponse", mock.Anything, mock.Anything).
		Return(&api.DeleteApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, nil)
}

func mockClient(t *testing.T, tokenURL, jwksURI *string) *eamoidcmocks.Client {
	t.Helper()
	clientMock := eamoidcmocks.NewClient(t)
	clientMock.On("GetTokenEndpoint", mock.Anything).Return(tokenURL, nil)
	clientMock.On("GetJWKSURI", mock.Anything).Return(jwksURI, nil)
	return clientMock
}

func mockGetTokenEndpoint(t *testing.T, tokenURL *string) *eamoidcmocks.Client {
	t.Helper()
	clientMock := eamoidcmocks.NewClient(t)
	clientMock.On("GetTokenEndpoint", mock.Anything).Return(tokenURL, nil)
	return clientMock
}
