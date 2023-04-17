package ias

import (
	"context"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/api"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

type Client interface {
	CreateApplication(ctx context.Context, name string) (Application, error)
}

func NewIasClient(iasTenantUrl, clientId, clientSecret string) (Client, error) {

	basicAuthProvider, err := securityprovider.NewSecurityProviderBasicAuth(clientId, clientSecret)
	if err != nil {
		return nil, err
	}

	applicationsEndpointUrl := fmt.Sprintf("%s/%s", iasTenantUrl, "Applications/v1/")
	apiClient, err := api.NewClientWithResponses(applicationsEndpointUrl, api.WithRequestEditorFn(basicAuthProvider.Intercept))
	if err != nil {
		return nil, err
	}

	return client{
		api:       apiClient,
		tenantUrl: iasTenantUrl,
	}, nil
}

type client struct {
	api       *api.ClientWithResponses
	tenantUrl string
}

// CreateApplication creates an application in IAS
func (c client) CreateApplication(ctx context.Context, name string) (Application, error) {

	newApplication := newIasApplication(name)
	createAppResponse, err := c.api.CreateApplicationWithResponse(ctx, &api.CreateApplicationParams{}, newApplication)
	if err != nil {
		return Application{}, err
	}

	// TODO: Do we need handling for 429 (Too Many Requests) or other status codes?
	if createAppResponse.StatusCode() != http.StatusCreated {
		ctrl.Log.Error(err, "Failed to create application", "name", name, "statusCode", createAppResponse.StatusCode())
		return Application{}, fmt.Errorf("failed to create application")
	}

	appId, err := extractApplicationId(createAppResponse)
	if err != nil {
		return Application{}, err
	}
	ctrl.Log.Info("Created application", "name", name, "id", appId)

	createSecretResponse, err := c.api.CreateApiSecretWithResponse(ctx, appId, newSecretRequest())
	if err != nil {
		return Application{}, err
	}

	if createSecretResponse.StatusCode() != http.StatusCreated {
		ctrl.Log.Error(err, "Failed to create api secret", "id", appId, "statusCode", createSecretResponse.StatusCode())
		return Application{}, fmt.Errorf("failed to create api secret")
	}
	clientSecret := *createSecretResponse.JSON201.Secret

	// The client ID is generated only after an API secret is created, so we need to retrieve the application again to get the client ID.
	applicationResponse, err := c.api.GetApplicationWithResponse(ctx, appId, &api.GetApplicationParams{})
	if err != nil {
		return Application{}, err
	}

	if applicationResponse.StatusCode() != http.StatusOK {
		ctrl.Log.Error(err, "Failed to create api secret", "id", appId, "statusCode", createSecretResponse.StatusCode())
		return Application{}, fmt.Errorf("failed to create api secret")
	}
	clientId := *applicationResponse.JSON200.UrnSapIdentityApplicationSchemasExtensionSci10Authentication.ClientId

	return Application{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		TokenUrl:     fmt.Sprintf("%s/oauth/token?grant_type=client_credentials&client_id=%s", c.tenantUrl, clientId),
	}, nil
}

func extractApplicationId(createAppResponse *api.CreateApplicationResponse) (uuid.UUID, error) {
	// The application ID is only returned in the location header
	locationHeader := createAppResponse.HTTPResponse.Header.Get("Location")
	s := strings.Split(locationHeader, "/")
	appId := s[len(s)-1]

	return uuid.Parse(appId)
}

func newIasApplication(name string) api.Application {
	ssoType := api.OpenIdConnect
	return api.Application{
		Name: &name,
		Branding: &api.Branding{
			DisplayName: &name,
		},
		Schemas: &[]api.SchemasEnum{
			api.SchemasEnumUrnSapIdentityApplicationSchemasExtensionSci10Authentication,
		},
		UrnSapIdentityApplicationSchemasExtensionSci10Authentication: &api.AuthenticationSchema{
			SsoType: &ssoType,
		},
	}
}

func newSecretRequest() api.CreateApiSecretJSONRequestBody {
	description := "eventing-auth-manager"
	requestBody := api.CreateApiSecretJSONRequestBody{
		AuthorizationScopes: &[]api.AuthorizationScope{"manageApp", "oAuth", "manageUsers"},
		Description:         &description,
		// TODO: What expiration should we set expiration? Current implementation relies on default expiration of 100 years
	}
	return requestBody
}
