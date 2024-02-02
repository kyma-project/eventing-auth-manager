package ias

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/api"
	"github.com/kyma-project/eventing-auth-manager/internal/ias/internal/oidc"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Client interface {
	CreateApplication(ctx context.Context, name string) (Application, error)
	DeleteApplication(ctx context.Context, name string) error
	GetCredentials() *Credentials
}

var NewClient = func(iasTenantUrl, user, password string) (Client, error) {

	basicAuthProvider, err := securityprovider.NewSecurityProviderBasicAuth(user, password)
	if err != nil {
		return nil, err
	}

	applicationsEndpointUrl := fmt.Sprintf("%s/Applications/v1/", iasTenantUrl)
	apiClient, err := api.NewClientWithResponses(applicationsEndpointUrl, api.WithRequestEditorFn(basicAuthProvider.Intercept))
	if err != nil {
		return nil, err
	}

	const timeout = time.Second * 5
	oidcHttpClient := &http.Client{
		Timeout: timeout,
	}

	return &client{
		api:         apiClient,
		oidcClient:  oidc.NewOidcClient(oidcHttpClient, iasTenantUrl),
		credentials: &Credentials{URL: iasTenantUrl, Username: user, Password: password},
	}, nil
}

type client struct {
	api        api.ClientWithResponsesInterface
	oidcClient oidc.Client
	// The token URL of the IAS client. Since this URL should only change when the tenant changes and this will lead to the initialization of
	// a new client, we can cache the URL to avoid an additional request at each application creation.
	tokenUrl *string
	// The jwks URI of the IAS client. Since this URI should only change when the tenant changes and this will lead to the initialization of
	// a new client, we can cache the URI to avoid an additional request at each application creation.
	jwksURI     *string
	credentials *Credentials
}

func (c *client) GetCredentials() *Credentials {
	if c.credentials == nil {
		c.credentials = &Credentials{}
	}
	return c.credentials
}

// CreateApplication creates an application in IAS. This function is not idempotent, because if an application with the specified
// name already exists, it will be deleted and recreated.
func (c *client) CreateApplication(ctx context.Context, name string) (Application, error) {
	existingApp, err := c.getApplicationByName(ctx, name)
	if err != nil {
		return Application{}, err
	}

	// To simplify the logic, if an application with this name already exists, we always delete the application and create
	// a new one, otherwise we would have to check where the application creation failed and continue at this point.
	if existingApp != nil {
		res, err := c.api.DeleteApplicationWithResponse(ctx, *existingApp.Id)
		if err != nil {
			return Application{}, err
		}
		if res.StatusCode() != http.StatusOK {
			ctrl.Log.Error(err, "Failed to delete existing application", "id", *existingApp.Id, "statusCode", res.StatusCode())
			return Application{}, errors.New("failed to delete existing application before creation")
		}
	}

	appId, err := c.createNewApplication(ctx, name)
	if err != nil {
		return Application{}, err
	}
	ctrl.Log.Info("Created application", "name", name, "id", appId)

	clientSecret, err := c.createSecret(ctx, appId)
	if err != nil {
		return Application{}, err
	}

	clientId, err := c.getClientId(ctx, appId)
	if err != nil {
		return Application{}, err
	}

	// Since the token url is not part of the application response, we have to fetch it from the OIDC configuration.
	tokenUrl, err := c.GetTokenUrl(ctx)
	if err != nil {
		return Application{}, err
	}

	// Since the jwks URI is not part of the application response, we have to fetch it from the OIDC configuration.
	jwksURI, err := c.GetJWKSURI(ctx)
	if err != nil {
		return Application{}, err
	}

	return NewApplication(appId.String(), *clientId, *clientSecret, *tokenUrl, *jwksURI), nil
}

func (c *client) GetTokenUrl(ctx context.Context) (*string, error) {
	if c.tokenUrl == nil {
		tokenEndpoint, err := c.oidcClient.GetTokenEndpoint(ctx)
		if err != nil {
			return nil, err
		}
		if tokenEndpoint == nil {
			return nil, errors.New("failed to fetch token url")
		}

		c.tokenUrl = tokenEndpoint
	}

	return c.tokenUrl, nil
}

func (c *client) GetJWKSURI(ctx context.Context) (*string, error) {
	if c.jwksURI == nil {
		jwksURI, err := c.oidcClient.GetJWKSURI(ctx)
		if err != nil {
			return nil, err
		}
		if jwksURI == nil {
			return nil, errors.New("failed to fetch jwks uri")
		}

		c.jwksURI = jwksURI
	}

	return c.jwksURI, nil
}

// DeleteApplication deletes an application in IAS. If the application does not exist, this function does nothing.
func (c *client) DeleteApplication(ctx context.Context, name string) error {
	existingApp, err := c.getApplicationByName(ctx, name)
	if err != nil {
		return err
	}
	if existingApp == nil {
		return nil
	}

	return c.deleteApplication(ctx, *existingApp.Id)
}

func (c *client) getApplicationByName(ctx context.Context, name string) (*api.ApplicationResponse, error) {
	appsFilter := fmt.Sprintf("name eq %s", name)
	res, err := c.api.GetAllApplicationsWithResponse(ctx, &api.GetAllApplicationsParams{Filter: &appsFilter})
	if err != nil {
		return nil, err
	}

	// This is not documented in the API, but the actual API returned 404 if no applications were found.
	if res.StatusCode() == http.StatusNotFound {
		return nil, nil //nolint:nilnil
	}

	if res.StatusCode() != http.StatusOK {
		ctrl.Log.Error(err, "Failed to fetch existing applications filtered by name", "name", name, "statusCode", res.StatusCode())
		return nil, errors.New("failed to fetch existing applications")
	}

	if res.JSON200.Applications != nil {
		switch len(*res.JSON200.Applications) {
		// Since the handling of the 404 status is not documented, we also handle the case where no more applications are found,
		// because we do not know what the expected behavior should be.
		case 0:
			return nil, nil //nolint:nilnil
		case 1:
			return &(*res.JSON200.Applications)[0], nil
		default:
			return nil, fmt.Errorf("found multiple applications with the same name %s", name)
		}
	}
	return nil, nil //nolint:nilnil
}

func (c *client) createNewApplication(ctx context.Context, name string) (uuid.UUID, error) {
	newApplication := newIasApplication(name)
	res, err := c.api.CreateApplicationWithResponse(ctx, &api.CreateApplicationParams{}, newApplication)
	if err != nil {
		return uuid.UUID{}, err
	}

	if res.StatusCode() != http.StatusCreated {
		ctrl.Log.Error(err, "Failed to create application", "name", name, "statusCode", res.StatusCode())
		return uuid.UUID{}, errors.New("failed to create application")
	}

	return extractApplicationId(res)
}

func (c *client) createSecret(ctx context.Context, appId uuid.UUID) (*string, error) {
	res, err := c.api.CreateApiSecretWithResponse(ctx, appId, newSecretRequest())
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusCreated {
		ctrl.Log.Error(err, "Failed to create api secret", "id", appId, "statusCode", res.StatusCode())
		return nil, errors.New("failed to create api secret")
	}

	return res.JSON201.Secret, nil
}

func (c *client) getClientId(ctx context.Context, appId uuid.UUID) (*string, error) {
	// The client ID is generated only after an API secret is created, so we need to retrieve the application again to get the client ID.
	applicationResponse, err := c.api.GetApplicationWithResponse(ctx, appId, &api.GetApplicationParams{})
	if err != nil {
		return nil, err
	}

	if applicationResponse.StatusCode() != http.StatusOK {
		ctrl.Log.Error(err, "Failed to retrieve client ID", "id", appId, "statusCode", applicationResponse.StatusCode())
		return nil, errors.New("failed to retrieve client ID")
	}
	return applicationResponse.JSON200.UrnSapIdentityApplicationSchemasExtensionSci10Authentication.ClientId, nil
}

func (c *client) deleteApplication(ctx context.Context, id uuid.UUID) error {
	res, err := c.api.DeleteApplicationWithResponse(ctx, id)
	if err != nil {
		return err
	}

	// This is not documented in the API, but the actual API returned 404 if no application is found for the given ID .
	if res.StatusCode() == http.StatusNotFound {
		return nil
	}

	if res.StatusCode() != http.StatusOK {
		ctrl.Log.Error(err, "Failed to delete application", "id", id, "statusCode", res.StatusCode())
		return errors.New("failed to delete application")
	}

	return nil
}

func extractApplicationId(createAppResponse *api.CreateApplicationResponse) (uuid.UUID, error) {
	// The application ID is only returned as the last part in the location header
	locationHeader := createAppResponse.HTTPResponse.Header.Get("Location")
	s := strings.Split(locationHeader, "/")
	appId := s[len(s)-1]

	parsedAppId, err := uuid.Parse(appId)
	if err != nil {
		return parsedAppId, errors.Wrap(err, "failed to retrieve application ID from header")
	}
	return parsedAppId, nil
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
	d := "eventing-auth-manager"
	requestBody := api.CreateApiSecretJSONRequestBody{
		AuthorizationScopes: &[]api.AuthorizationScope{"oAuth"},
		Description:         &d,
	}
	return requestBody
}
