package ias

import (
	"context"
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
	tests := []struct {
		name          string
		givenApiMock  func() *mocks.ClientWithResponsesInterface
		assertCalls   func(*testing.T, *mocks.ClientWithResponsesInterface)
		expectedApp   Application
		expectedError error
	}{
		{
			name: "should create new application when fetching existing applications returns status 200 and no applications",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockNoApplicationExists(&clientMock)

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponse(&clientMock, appId)

				d := "eventing-auth-manager"
				secretRequest := api.CreateApiSecretJSONRequestBody{
					AuthorizationScopes: &[]api.AuthorizationScope{"oAuth"},
					Description:         &d,
				}

				clientSecretMock := "clientSecretMock"
				clientMock.On("CreateApiSecretWithResponse", mock.Anything, appId, secretRequest).
					Return(&api.CreateApiSecretResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusCreated,
						},
						JSON201: &api.ApiSecretResponse{
							Secret: &clientSecretMock,
						},
					}, nil)

				clientIdMock := "clientIdMock"
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
				return &clientMock
			},
			expectedApp:   NewApplication("clientIdMock", "clientSecretMock", "https://test.com"),
			expectedError: nil,
		},
		{
			name: "should create new application when fetching existing applications returns status 404",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				clientMock.On("GetAllApplicationsWithResponse", mock.Anything, mock.Anything).
					Return(&api.GetAllApplicationsResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusNotFound,
						},
					}, nil)

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponse(&clientMock, appId)

				d := "eventing-auth-manager"
				secretRequest := api.CreateApiSecretJSONRequestBody{
					AuthorizationScopes: &[]api.AuthorizationScope{"oAuth"},
					Description:         &d,
				}

				clientSecretMock := "clientSecretMock"
				clientMock.On("CreateApiSecretWithResponse", mock.Anything, appId, secretRequest).
					Return(&api.CreateApiSecretResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusCreated,
						},
						JSON201: &api.ApiSecretResponse{
							Secret: &clientSecretMock,
						},
					}, nil)

				clientIdMock := "clientIdMock"
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
				return &clientMock
			},
			expectedApp:   NewApplication("clientIdMock", "clientSecretMock", "https://test.com"),
			expectedError: nil,
		},
		{
			name: "should recreate application when application already exists",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				existingAppId := uuid.MustParse("5ab797c0-80a0-4ca4-ad7f-50a0f40231d6")
				mockApplicationExists(&clientMock, existingAppId)

				clientMock.On("DeleteApplicationWithResponse", mock.Anything, existingAppId).
					Return(&api.DeleteApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusOK,
						},
					}, nil)

				newAppId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponse(&clientMock, newAppId)

				clientSecretMock := "clientSecretMock"
				clientMock.On("CreateApiSecretWithResponse", mock.Anything, newAppId, mock.Anything).
					Return(&api.CreateApiSecretResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusCreated,
						},
						JSON201: &api.ApiSecretResponse{
							Secret: &clientSecretMock,
						},
					}, nil)

				clientIdMock := "clientIdMock"
				clientMock.On("GetApplicationWithResponse", mock.Anything, newAppId, mock.Anything).
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
				return &clientMock
			},
			expectedApp:   NewApplication("clientIdMock", "clientSecretMock", "https://test.com"),
			expectedError: nil,
		},
		{
			name: "should return an error when multiple applications exist for the given name",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				appId2 := uuid.MustParse("41de6fec-e0fc-47d7-b35c-3b19c4927e4f")

				clientMock.On("GetAllApplicationsWithResponse", mock.Anything, mock.Anything).
					Return(&api.GetAllApplicationsResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusOK,
						},
						JSON200: &api.ApplicationsResponse{
							Applications: &[]api.ApplicationResponse{
								{
									Id: &appId,
								}, {
									Id: &appId2,
								},
							},
						},
					}, nil)

				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("found multiple applications with the same name Test-App-Name"),
		},
		{
			name: "should return error when application ID can't be retrieved from location header",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {

				clientMock := mocks.ClientWithResponsesInterface{}
				mockNoApplicationExists(&clientMock)

				clientMock.On("CreateApplicationWithResponse", mock.Anything, mock.Anything, newIasApplication("Test-App-Name")).
					Return(&api.CreateApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusCreated,
							Header: map[string][]string{
								"Location": {fmt.Sprintf("https://test.com/v1/Applications/%s", "non-uuid-application-id")},
							},
						},
					}, nil)

				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("failed to retrieve application ID from header: invalid UUID length: 23"),
		},
		{
			name: "should return error when fetching existing application failed",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				clientMock.On("GetAllApplicationsWithResponse", mock.Anything, mock.Anything).
					Return(&api.GetAllApplicationsResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}, nil)

				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("failed to fetch existing applications"),
		},
		{
			name: "should return error when application exists and deletion failed",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockApplicationExists(&clientMock, appId)

				clientMock.On("DeleteApplicationWithResponse", mock.Anything, mock.Anything, mock.Anything).
					Return(&api.DeleteApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}, nil)

				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("failed to delete application"),
		},
		{
			name: "should return error when application is not created",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockNoApplicationExists(&clientMock)

				clientMock.On("CreateApplicationWithResponse", mock.Anything, mock.Anything, mock.Anything).
					Return(&api.CreateApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}, nil)

				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("failed to create application"),
		},
		{
			name: "should return error when secret is not created",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockNoApplicationExists(&clientMock)

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponse(&clientMock, appId)

				clientMock.On("CreateApiSecretWithResponse", mock.Anything, mock.Anything, mock.Anything).
					Return(&api.CreateApiSecretResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}, nil)

				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("failed to create api secret"),
		},
		{
			name: "should return error when client id wasn't fetched",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockNoApplicationExists(&clientMock)

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockCreateApplicationWithResponse(&clientMock, appId)

				clientSecretMock := "clientSecretMock"
				clientMock.On("CreateApiSecretWithResponse", mock.Anything, appId, mock.Anything).
					Return(&api.CreateApiSecretResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusCreated,
						},
						JSON201: &api.ApiSecretResponse{
							Secret: &clientSecretMock,
						},
					}, nil)

				clientMock.On("GetApplicationWithResponse", mock.Anything, appId, mock.Anything).
					Return(&api.GetApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}, nil)
				return &clientMock
			},
			expectedApp:   Application{},
			expectedError: fmt.Errorf("failed to retrieve client ID"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			apiMock := test.givenApiMock()

			client := client{
				api:       apiMock,
				tenantUrl: "https://test.com",
			}

			// when
			app, err := client.CreateApplication(context.TODO(), "Test-App-Name")

			// then
			require.Equal(t, test.expectedApp, app)

			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			if test.assertCalls != nil {
				test.assertCalls(t, apiMock)
			} else {
				apiMock.AssertExpectations(t)
			}

		})
	}
}

func Test_DeleteApplication(t *testing.T) {
	tests := []struct {
		name          string
		givenApiMock  func() *mocks.ClientWithResponsesInterface
		expectedError error
	}{
		{
			name: "should delete application",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockApplicationExists(&clientMock, appId)

				clientMock.On("DeleteApplicationWithResponse", mock.Anything, appId).
					Return(&api.DeleteApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusOK,
						},
					}, nil)

				return &clientMock
			},
			expectedError: nil,
		},
		{
			name: "should return error when application is not deleted",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				appId := uuid.MustParse("90764f89-f041-4ccf-8da9-7a7c2d60d7fc")
				mockApplicationExists(&clientMock, appId)

				clientMock.On("DeleteApplicationWithResponse", mock.Anything, appId).
					Return(&api.DeleteApplicationResponse{
						HTTPResponse: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}, nil)

				return &clientMock
			},
			expectedError: fmt.Errorf("failed to delete application"),
		},
		{
			name: "should not return an error when application doesn't exist",
			givenApiMock: func() *mocks.ClientWithResponsesInterface {
				clientMock := mocks.ClientWithResponsesInterface{}

				mockNoApplicationExists(&clientMock)

				return &clientMock
			},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			apiMock := test.givenApiMock()

			client := client{
				api:       apiMock,
				tenantUrl: "https://test.com",
			}

			// when
			err := client.DeleteApplication(context.TODO(), "Test-App-Name")

			// then
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			apiMock.AssertExpectations(t)
		})
	}
}

func mockNoApplicationExists(clientMock *mocks.ClientWithResponsesInterface) {
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

func mockApplicationExists(clientMock *mocks.ClientWithResponsesInterface, appId uuid.UUID) {
	appsFilter := "name eq Test-App-Name"
	clientMock.On("GetAllApplicationsWithResponse", mock.Anything, &api.GetAllApplicationsParams{Filter: &appsFilter}).
		Return(&api.GetAllApplicationsResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &api.ApplicationsResponse{
				Applications: &[]api.ApplicationResponse{
					{
						Id: &appId,
					},
				},
			},
		}, nil)
}

func mockCreateApplicationWithResponse(clientMock *mocks.ClientWithResponsesInterface, appId uuid.UUID) {
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
