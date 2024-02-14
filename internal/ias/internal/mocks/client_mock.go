package mocks

import (
	"context"
	"reflect"

	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kpkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MockClient struct {
	MockFunction func() error
	MockSecret   *kcorev1.Secret
}

func (m MockClient) Get(ctx context.Context, key kpkgclient.ObjectKey, out kpkgclient.Object, opts ...kpkgclient.GetOption) error {
	outVal := reflect.ValueOf(out)
	objVal := reflect.ValueOf(m.MockSecret)
	reflect.Indirect(outVal).Set(reflect.Indirect(objVal))
	return m.MockFunction()
}

func (m MockClient) List(ctx context.Context, list kpkgclient.ObjectList, opts ...kpkgclient.ListOption) error {
	return nil
}

func (m MockClient) Create(ctx context.Context, obj kpkgclient.Object, opts ...kpkgclient.CreateOption) error {
	return nil
}

func (m MockClient) Delete(ctx context.Context, obj kpkgclient.Object, opts ...kpkgclient.DeleteOption) error {
	return nil
}

func (m MockClient) Update(ctx context.Context, obj kpkgclient.Object, opts ...kpkgclient.UpdateOption) error {
	return nil
}

func (m MockClient) Patch(ctx context.Context, obj kpkgclient.Object, patch kpkgclient.Patch, opts ...kpkgclient.PatchOption) error {
	return nil
}

func (m MockClient) DeleteAllOf(ctx context.Context, obj kpkgclient.Object, opts ...kpkgclient.DeleteAllOfOption) error {
	return nil
}

func (m MockClient) Status() kpkgclient.SubResourceWriter {
	return nil
}

func (m MockClient) SubResource(subResource string) kpkgclient.SubResourceClient {
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
