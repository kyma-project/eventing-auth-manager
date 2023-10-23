package skr

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_NewClient(t *testing.T) {
	type args struct {
		k8sClient    ctrlclient.Client
		skrClusterId string
	}
	tests := []struct {
		name      string
		args      args
		wantError error
	}{
		{
			name: "should return error when secret with kubeconfig is not found",
			args: args{
				k8sClient:    fake.NewClientBuilder().Build(),
				skrClusterId: "test",
			},
			wantError: errors.New("secrets \"kubeconfig-test\" not found"),
		},
		{
			name: "should return error when secret doesn't contain config key",
			args: args{
				k8sClient:    fake.NewClientBuilder().WithObjects(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "kubeconfig-test", Namespace: KcpNamespace}}).Build(),
				skrClusterId: "test",
			},
			wantError: errors.New("failed to find SKR cluster kubeconfig in secret kubeconfig-test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			_, err := NewClient(tt.args.k8sClient, tt.args.skrClusterId)

			// then
			require.Error(t, err)
			require.EqualError(t, tt.wantError, err.Error())
		})
	}
}

func Test_client_DeleteSecret(t *testing.T) {
	type fields struct {
		k8sClient ctrlclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "should return no error when secret exists",
			fields: fields{
				k8sClient: fake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ApplicationSecretName,
							Namespace: ApplicationSecretNamespace,
						},
					}).Build(),
			},
		},
		{
			name: "should return no error when secret does not exist",
			fields: fields{
				k8sClient: fake.NewClientBuilder().Build(),
			},
		},
		{
			name: "should return error when fetching secret",
			fields: fields{
				k8sClient: errorFakeClient{
					Client:     fake.NewClientBuilder().Build(),
					errorOnGet: errors.New("error on getting secret"),
				},
			},
			wantErr: errors.New("error on getting secret"),
		},
		{
			name: "should ignore NotFound error when fetching secret",
			fields: fields{
				k8sClient: errorFakeClient{
					Client:     fake.NewClientBuilder().Build(),
					errorOnGet: apierrors.NewNotFound(v1.Resource("secret"), ApplicationSecretName),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				k8sClient: tt.fields.k8sClient,
			}

			err := c.DeleteSecret(context.TODO())

			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_client_HasApplicationSecret(t *testing.T) {
	type fields struct {
		k8sClient ctrlclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr error
	}{
		{
			name: "should return false when secret is not found",
			fields: fields{
				k8sClient: fake.NewClientBuilder().Build(),
			},
			want: false,
		},
		{
			name: "should return true when secret is found",
			fields: fields{
				k8sClient: fake.NewClientBuilder().WithObjects(
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ApplicationSecretName,
							Namespace: ApplicationSecretNamespace,
						},
					}).Build(),
			},
			want: true,
		},
		{
			name: "should return error when fetching secret",
			fields: fields{
				k8sClient: errorFakeClient{
					Client:     fake.NewClientBuilder().Build(),
					errorOnGet: errors.New("error on getting secret"),
				},
			},
			want:    false,
			wantErr: errors.New("error on getting secret"),
		},
		{
			name: "should ignore NotFound error when fetching secret",
			fields: fields{
				k8sClient: errorFakeClient{
					Client:     fake.NewClientBuilder().Build(),
					errorOnGet: apierrors.NewNotFound(v1.Resource("secret"), ApplicationSecretName),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				k8sClient: tt.fields.k8sClient,
			}

			got, err := c.HasApplicationSecret(context.TODO())

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

type errorFakeClient struct {
	ctrlclient.Client
	errorOnGet error
}

func (e errorFakeClient) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	if e.errorOnGet == nil {
		return e.Client.Get(ctx, key, obj, opts...)
	}
	return e.errorOnGet
}
