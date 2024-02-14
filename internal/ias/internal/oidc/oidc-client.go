package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

//go:generate mockery --name=Client --outpkg=mocks --case=underscore
type Client interface {
	GetTokenEndpoint(ctx context.Context) (*string, error)
	GetJWKSURI(ctx context.Context) (*string, error)
}

type wellKnown struct {
	TokenEndpoint *string `json:"token_endpoint,omitempty"`
	JWKSURI       *string `json:"jwks_uri,omitempty"`
}

type client struct {
	domainURL  string
	httpClient *http.Client
}

// NewOidcClient returns a new OIDC client. The domain URL is used to get the OIDC configuration for a specific tenant, e.g. 'https://some-tenant.accounts400.ondemand.com'.
func NewOidcClient(h *http.Client, domainURL string) Client {
	return client{
		domainURL:  domainURL,
		httpClient: h,
	}
}

// GetTokenEndpoint returns the OIDC token endpoint for a specific tenant.
func (c client) GetTokenEndpoint(ctx context.Context) (*string, error) {
	w, err := c.getWellKnown(ctx)
	if err != nil {
		return nil, err
	}

	return w.TokenEndpoint, nil
}

// GetJWKSURI returns the OIDC jwks uri for a specific tenant.
func (c client) GetJWKSURI(ctx context.Context) (*string, error) {
	w, err := c.getWellKnown(ctx)
	if err != nil {
		return nil, err
	}

	return w.JWKSURI, nil
}

func (c client) getWellKnown(ctx context.Context) (wellKnown, error) {
	url := fmt.Sprintf("%s/.well-known/openid-configuration", c.domainURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return wellKnown{}, err
	}

	body, err := c.do(req)
	if err != nil {
		return wellKnown{}, err
	}

	w := wellKnown{}
	if err := json.Unmarshal(body, &w); err != nil {
		return wellKnown{}, err
	}

	return w, nil
}

func (c client) do(req *http.Request) ([]byte, error) {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code %d", res.StatusCode)
	}

	if res.Body != nil {
		defer func() { _ = res.Body.Close() }()
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	return body, nil
}
