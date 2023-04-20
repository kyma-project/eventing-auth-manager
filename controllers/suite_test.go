/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/eventing-auth-manager/controllers"
	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	operatorv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	defaultTimeout = time.Second * 5
	namespace      = "test-namespace"
	kymaNs         = "kyma-system"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(specCtx SpecContext) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	Expect(operatorv1alpha1.AddToScheme(scheme.Scheme)).Should(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
		Spec:       corev1.NamespaceSpec{},
	}
	Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())

	kymaNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: kymaNs},
		Spec:       corev1.NamespaceSpec{},
	}
	Expect(k8sClient.Create(context.TODO(), kymaNs)).Should(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	reconciler := controllers.NewEventingAuthReconciler(mgr.GetClient(), mgr.GetScheme(), getIasTestClient())

	Expect(reconciler.SetupWithManager(mgr)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).Should(Succeed())
	}()

}, NodeTimeout(60*time.Second))

var _ = AfterSuite(func() {
	cancel()
	By("Tearing down the test environment")
	err := testEnv.Stop()

	// Suggested workaround for timeout issue https://github.com/kubernetes-sigs/controller-runtime/issues/1571#issuecomment-1005575071
	if err != nil {
		time.Sleep(4 * time.Second)
	}
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

})

func getIasTestClient() ias.Client {
	url := os.Getenv("TEST_EVENTING_AUTH_IAS_URL")
	user := os.Getenv("TEST_EVENTING_AUTH_IAS_USER")
	pw := os.Getenv("TEST_EVENTING_AUTH_IAS_PASSWORD")

	if url != "" && user != "" && pw != "" {
		iasClient, err := ias.NewIasClient(url, user, pw)
		Expect(err).NotTo(HaveOccurred())
		return iasClient
	} else {
		return iasClientStub{}
	}
}

type iasClientStub struct {
}

func (i iasClientStub) CreateApplication(_ context.Context, name string) (ias.Application, error) {
	return ias.NewApplication(fmt.Sprintf("id-for-%s", name), fmt.Sprintf("client-id-for-%s", name), "test-client-secret", "https://test-token-url.com"), nil
}

func (i iasClientStub) DeleteApplication(_ context.Context, _ string) error {
	return nil
}
