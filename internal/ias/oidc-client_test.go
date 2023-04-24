package ias

import (
	"bytes"
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"k8s.io/client-go/rest/fake"
	"k8s.io/utils/pointer"
	"net/http"
	"net/url"
	"testing"
)

const oidcConfigMock = `{"token_endpoint":"https://test.com/token"}`

func Test_oidcClient_do(t *testing.T) {
	type fields struct {
		baseUrl    string
		httpClient *http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr error
	}{
		{
			name: "should return error when response status code is not 200",
			fields: fields{
				baseUrl: "http://localhost",
				httpClient: fake.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
					}, nil
				}),
			},
			want:    nil,
			wantErr: errors.New("unexpected status code 500"),
		},
		{
			name: "should return empty body when response body is nil",
			fields: fields{
				httpClient: mockHttpClientResponseOk(nil),
			},
			want: []byte{},
		},
		{
			name: "should return body when response body is not nil",
			fields: fields{
				httpClient: mockHttpClientResponseOk([]byte("body value")),
			},
			want: []byte("body value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c := oidcClient{
				httpClient: tt.fields.httpClient,
			}

			urlMock, _ := url.Parse("http://localhost")

			// when
			got, err := c.do(&http.Request{
				URL: urlMock,
			})

			// then
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func Test_oidcClient_getTokenUrl(t *testing.T) {
	type fields struct {
		baseUrl    string
		httpClient *http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    *string
		wantErr error
	}{
		{
			name: "should return token url from well known",
			fields: fields{
				baseUrl:    "http://base-url-from-client.com",
				httpClient: mockHttpClientResponseOk([]byte(oidcConfigMock)),
			},
			want: pointer.String("https://test.com/token"),
		},
		{
			name: "should return nil when well known is not well structured",
			fields: fields{
				baseUrl:    "http://base-url-from-client.com",
				httpClient: mockHttpClientResponseOk([]byte("{}")),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c := oidcClient{
				baseUrl:    tt.fields.baseUrl,
				httpClient: tt.fields.httpClient,
			}

			// when
			got, err := c.GetTokenUrl(context.TODO())

			// then
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func Test_oidcClient_getWellKnown(t *testing.T) {
	type fields struct {
		baseUrl    string
		httpClient *http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    wellKnown
		wantErr error
	}{
		{
			name: "should return error when response is no json",
			fields: fields{
				baseUrl:    "http://localhost.com",
				httpClient: mockHttpClientResponseOk([]byte("invalid json")),
			},
			want:    wellKnown{},
			wantErr: errors.New("invalid character 'i' looking for beginning of value"),
		},
		{
			name: "should return well known by using base url from client and well-known OIDC config path for request",
			fields: fields{
				baseUrl: "http://base-url-from-client.com",
				httpClient: fake.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
					require.Equal(t, http.MethodGet, request.Method)
					require.Equal(t, "base-url-from-client.com", request.URL.Host)
					require.Equal(t, "/.well-known/openid-configuration", request.URL.Path)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(oidcConfigMock))),
					}, nil
				}),
			},
			want: wellKnown{TokenEndpoint: pointer.String("https://test.com/token")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			c := oidcClient{
				baseUrl:    tt.fields.baseUrl,
				httpClient: tt.fields.httpClient,
			}

			// when
			got, err := c.getWellKnown(context.TODO())

			// then
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func mockHttpClientResponseOk(body []byte) *http.Client {
	return fake.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil
	})
}
