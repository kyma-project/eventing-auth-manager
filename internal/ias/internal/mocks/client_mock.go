package mocks

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctlrClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MockClient struct {
	MockFunction func() error
	MockSecret   *corev1.Secret
}

func (m MockClient) Get(ctx context.Context, key ctlrClient.ObjectKey, out ctlrClient.Object, opts ...ctlrClient.GetOption) error {
	outVal := reflect.ValueOf(out)
	objVal := reflect.ValueOf(m.MockSecret)
	reflect.Indirect(outVal).Set(reflect.Indirect(objVal))
	return m.MockFunction()
}

func (m MockClient) List(ctx context.Context, list ctlrClient.ObjectList, opts ...ctlrClient.ListOption) error {
	return nil
}

func (m MockClient) Create(ctx context.Context, obj ctlrClient.Object, opts ...ctlrClient.CreateOption) error {
	return nil
}

func (m MockClient) Delete(ctx context.Context, obj ctlrClient.Object, opts ...ctlrClient.DeleteOption) error {
	return nil
}

func (m MockClient) Update(ctx context.Context, obj ctlrClient.Object, opts ...ctlrClient.UpdateOption) error {
	return nil
}

func (m MockClient) Patch(ctx context.Context, obj ctlrClient.Object, patch ctlrClient.Patch, opts ...ctlrClient.PatchOption) error {
	return nil
}

func (m MockClient) DeleteAllOf(ctx context.Context, obj ctlrClient.Object, opts ...ctlrClient.DeleteAllOfOption) error {
	return nil
}

func (m MockClient) Status() ctlrClient.SubResourceWriter {
	return nil
}

func (m MockClient) SubResource(subResource string) ctlrClient.SubResourceClient {
	return nil
}

func (m MockClient) Scheme() *runtime.Scheme {
	return nil
}

func (m MockClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (m MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return false, nil
}
