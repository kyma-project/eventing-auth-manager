package ias

type Client interface {
	CreateApplication(name string) (Application, error)
}

func NewIasClient(url, clientId, clientSecret string) Client {
	return client{
		url:          url,
		clientId:     clientId,
		clientSecret: clientSecret,
	}
}

type client struct {
	url          string
	clientId     string
	clientSecret string
}

// CreateApplication creates an application in IAS
func (c client) CreateApplication(_ string) (Application, error) {
	return Application{
		clientId:     "",
		clientSecret: "",
		tokenUrl:     "",
	}, nil
}
