package ias

import "fmt"

type Application struct {
	ClientId     string
	ClientSecret string
	TokenUrl     string
}

func NewApplication(clientId, clientSecret, tenantUrl string) Application {
	return Application{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		TokenUrl:     fmt.Sprintf("%s/oauth2/token?grant_type=client_credentials&client_id=%s", tenantUrl, clientId),
	}
}
