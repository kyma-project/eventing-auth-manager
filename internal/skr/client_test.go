package skr

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_NewClient(t *testing.T) {
	type args struct {
		k8sClient       ctrlclient.Client
		targetClusterId string
	}
	tests := []struct {
		name       string
		args       args
		wantClient bool
		wantError  error
	}{
		{
			name: "should return error when secret with kubeconfig is not found",
			args: args{
				k8sClient:       fake.NewClientBuilder().Build(),
				targetClusterId: "test",
			},
			wantClient: false,
			wantError:  errors.New("secrets \"kubeconfig-test\" not found"),
		},
		{
			name: "should return error when secret doesn't contain config key",
			args: args{
				k8sClient:       fake.NewClientBuilder().WithObjects(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "kubeconfig-test", Namespace: "kcp-system"}}).Build(),
				targetClusterId: "test",
			},
			wantClient: false,
			wantError:  errors.New("failed to find target cluster kubeconfig in secret kubeconfig-test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got, err := NewClient(tt.args.k8sClient, tt.args.targetClusterId)

			// then
			if tt.wantError != nil {
				require.Error(t, err)
				require.EqualError(t, tt.wantError, err.Error())
			} else {
				require.NoError(t, err)
			}

			if tt.wantClient {
				require.NotNil(t, got)
			}
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
							Name:      "eventing-webhook-auth",
							Namespace: "kyma-system",
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
							Name:      "eventing-webhook-auth",
							Namespace: "kyma-system",
						},
					}).Build(),
			},
			want: true,
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
