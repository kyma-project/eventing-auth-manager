package controllers

import (
	"errors"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_getTargetClusterClient(t *testing.T) {
	type args struct {
		k8sClient       client.Client
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
			got, err := getTargetClusterClient(tt.args.k8sClient, tt.args.targetClusterId)

			// then
			if tt.wantError != nil {
				require.Error(t, tt.wantError)
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
