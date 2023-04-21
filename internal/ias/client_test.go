package ias

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/api"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func Test_CreateApplication(t *testing.T) {
	appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
	tests := []struct {
		name         string
		givenApiMock func() *mocks.ClientWithResponsesInterface
		assertCalls  func(*testing.T, *mocks.ClientWithResponsesInterface)
		wantApp      Application
		wantError    error
	}{
		{
			name: "should create new application when fetching existing applications returns status 200 and no applications",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appId.String())
				mockCreateApiSecretWithResponseStatusCreated(&clientMock, appId, "clientSecretMock")
				mockGetApplicationWithResponseStatusOK(&clientMock, appId, "clientIdMock")

				return &clientMock
			},
			wantApp: NewApplication(appId.String(), "clientIdMock", "clientSecretMock", "https://test.com"),
		},
		{
			name: "should create new application when fetching existing applications returns status 404",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")

				mockGetAllApplicationsWithResponseStatusNotFound(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appId.String())
				mockCreateApiSecretWithResponseStatusCreated(&clientMock, appId, "clientSecretMock")
				mockGetApplicationWithResponseStatusOK(&clientMock, appId, "clientIdMock")

				return &clientMock
			},
			wantApp: NewApplication(appId.String(), "clientIdMock", "clientSecretMock", "https://test.com"),
		},
		{
			name: "should recreate application when application already exists",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				existingAppId := uuid.MustParse("5ab797c0-80a0-4ca4-ad7f-50a0f40231d6")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, existingAppId)
				mockDeleteApplicationWithResponseStatusOk(&clientMock, existingAppId)

				newAppId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponseStatusCreated(&clientMock, newAppId.String())
				mockCreateApiSecretWithResponseStatusCreated(&clientMock, newAppId, "clientSecretMock")
				mockGetApplicationWithResponseStatusOK(&clientMock, newAppId, "clientIdMock")

				return &clientMock
			},
			wantApp: NewApplication(appId.String(), "clientIdMock", "clientSecretMock", "https://test.com"),
		},
		{
			name: "should return an error when multiple applications exist for the given name",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId2 := uuid.MustParse("41de6fec-e0fc-47d7-b35c-3b19c4927e4f")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appId, appId2)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("found multiple applications with the same name Test-App-Name"),
		},
		{
			name: "should return error when application ID can't be retrieved from location header",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {

				clientMock := mocks.ClientWithResponsesInterface{}
				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, "non-uuid-application-id")

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to retrieve application ID from header: invalid UUID length: 23"),
		},
		{
			name: "should return error when fetching existing application failed",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				mockGetAllApplicationsWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to fetch existing applications"),
		},
		{
			name: "should return error when application exists and deletion failed",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appId)
				mockDeleteApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to delete existing application before creation"),
		},
		{
			name: "should return error when application is not created",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to create application"),
		},
		{
			name: "should return error when secret is not created",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appId.String())
				mockCreateApiSecretWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to create api secret"),
		},
		{
			name: "should return error when client id wasn't fetched",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}
				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)
				mockCreateApplicationWithResponseStatusCreated(&clientMock, appId.String())
				mockCreateApiSecretWithResponseStatusCreated(&clientMock, appId, "clientSecretMock")
				mockGetApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantApp:   Application{},
			wantError: errors.New("failed to retrieve client ID"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiMock := tt.givenApiMock()

			client := client{
				api:       apiMock,
				tenantUrl: "https://test.com",
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

		})
	}
}

func Test_DeleteApplication(t *testing.T) {
	tests := []struct {
		name         string
		givenApiMock func() *mocks.ClientWithResponsesInterface
		wantError    error
	}{
		{
			name: "should delete application",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appId)
				mockDeleteApplicationWithResponseStatusOk(&clientMock, appId)

				return &clientMock
			},
		},
		{
			name: "should return error when application is not deleted",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appId)
				mockDeleteApplicationWithResponseStatusInternalServerError(&clientMock)

				return &clientMock
			},
			wantError: errors.New("failed to delete application"),
		},
		{
			name: "should not return an error when application doesn't exist",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockGetAllApplicationsWithResponseStatusOkEmptyResponse(&clientMock)

				return &clientMock
			},
		},
		{
			name: "should not return an error when app was found, but was already deleted",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockGetAllApplicationsWithResponseStatusOk(&clientMock, appId)
				mockDeleteApplicationWithResponseStatusNotFound(&clientMock)

				return &clientMock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			apiMock := tt.givenApiMock()

			client := client{
				api:       apiMock,
				tenantUrl: "https://test.com",
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
	for _, appId := range appIds {
		appResponses = append(appResponses, api.ApplicationResponse{
			Id: &appId,
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

func mockCreateApplicationWithResponseStatusCreated(clientMock *mocks.ClientWithResponsesInterface, appId string) {
	clientMock.On("CreateApplicationWithResponse", mock.Anything, mock.Anything, newIasApplication("Test-App-Name")).
		Return(&api.CreateApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusCreated,
				Header: map[string][]string{
					"Location": {fmt.Sprintf("https://test.com/v1/Applications/%s", appId)},
				},
			},
		}, nil)
}

func mockCreateApiSecretWithResponseStatusInternalServerError(clientMock *mocks.ClientWithResponsesInterface) {
	clientMock.On("CreateApiSecretWithResponse", mock.Anything, mock.Anything, mock.Anything).
		Return(&api.CreateApiSecretResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, nil)
}

func mockCreateApiSecretWithResponseStatusCreated(clientMock *mocks.ClientWithResponsesInterface, appId uuid.UUID, clientSecretMock string) {
	d := "eventing-auth-manager"
	secretRequest := api.CreateApiSecretJSONRequestBody{
		AuthorizationScopes: &[]api.AuthorizationScope{"oAuth"},
		Description:         &d,
	}
	clientMock.On("CreateApiSecretWithResponse", mock.Anything, appId, secretRequest).
		Return(&api.CreateApiSecretResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusCreated,
			},
			JSON201: &api.ApiSecretResponse{
				Secret: &clientSecretMock,
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

func mockGetApplicationWithResponseStatusOK(clientMock *mocks.ClientWithResponsesInterface, appId uuid.UUID, clientIdMock string) {
	clientMock.On("GetApplicationWithResponse", mock.Anything, appId, mock.Anything).
		Return(&api.GetApplicationResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &api.ApplicationResponse{
				UrnSapIdentityApplicationSchemasExtensionSci10Authentication: &api.AuthenticationSchema{
					ClientId: &clientIdMock,
				},
			},
		}, nil)
}

func mockDeleteApplicationWithResponseStatusOk(clientMock *mocks.ClientWithResponsesInterface, appId uuid.UUID) {
	clientMock.On("DeleteApplicationWithResponse", mock.Anything, appId).
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
