package ias

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

//go:generate mockery --name=OidcConfigurationClient --outpkg=mocks --case=underscore
type OidcConfigurationClient interface {
	GetTokenUrl(ctx context.Context) (*string, error)
}

type wellKnown struct {
	TokenEndpoint *string `json:"token_endpoint,omitempty"`
}

type oidcClient struct {
	baseUrl    string
	httpClient *http.Client
}

func newOidcClient(tenantUrl string) oidcClient {
	return oidcClient{
		baseUrl: tenantUrl,
		httpClient: &http.Client{
			Timeout: time.Second * 2,
		},
	}
}

func (c oidcClient) GetTokenUrl(ctx context.Context) (*string, error) {
	w, err := c.getWellKnown(ctx)
	if err != nil {
		return nil, err
	}

	return w.TokenEndpoint, nil
}

func (c oidcClient) getWellKnown(ctx context.Context) (wellKnown, error) {
	url := fmt.Sprintf("%s/.well-known/openid-configuration", c.baseUrl)
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

func (c oidcClient) do(req *http.Request) ([]byte, error) {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", res.StatusCode)
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
