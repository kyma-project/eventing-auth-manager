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
	"encoding/base64"
	"fmt"
	"github.com/kyma-project/eventing-auth-manager/controllers"
	"github.com/kyma-project/eventing-auth-manager/internal/ias"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
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
	"log"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	defaultTimeout = time.Second * 10
	namespace      = "test-namespace"
)

var (
	cfg                         *rest.Config
	k8sClient                   client.Client
	ctx                         context.Context
	cancel                      context.CancelFunc
	targetClusterK8sCfg         string
	targetClusterK8sClient      client.Client
	testEnv                     *envtest.Environment
	targetClusterTestEnv        *envtest.Environment
	originalNewIasClientFunc    func(iasTenantUrl, user, password string) (ias.Client, error)
	originalReadCredentialsFunc func(namespace, name string, k8sClient client.Client) (*ias.Credentials, error)
	iasUrl                      string
	iasUsername                 string
	iasPassword                 string
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(specCtx SpecContext) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	iasUrl = os.Getenv("TEST_EVENTING_AUTH_IAS_URL")
	iasUsername = os.Getenv("TEST_EVENTING_AUTH_IAS_USER")
	iasPassword = os.Getenv("TEST_EVENTING_AUTH_IAS_PASSWORD")

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error

	// Start Target Cluster
	targetClusterK8sClient, err = initTargetClusterConfig()
	Expect(err).NotTo(HaveOccurred())

	// In case we are using an existing cluster the Kyma-system namespace might already exist, so we need to guard against that.
	kymaNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	err = targetClusterK8sClient.Get(context.TODO(), client.ObjectKeyFromObject(kymaNs), &corev1.Namespace{})
	if err != nil {
		Expect(errors.IsNotFound(err)).To(BeTrue())
		err := targetClusterK8sClient.Create(context.TODO(), kymaNs)
		Expect(err).NotTo(HaveOccurred())
	}

	// Start the test cluster
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

	kcpNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kcp-system"},
		Spec:       corev1.NamespaceSpec{},
	}
	Expect(k8sClient.Create(context.TODO(), kcpNs)).Should(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	// mock ias.ReadCredentials function
	originalReadCredentialsFunc = ias.ReadCredentials
	ias.ReadCredentials = func(namespace, name string, k8sClient client.Client) (*ias.Credentials, error) {
		return &ias.Credentials{URL: iasUrl, Username: iasUsername, Password: iasPassword}, nil
	}

	if !existIasCreds() {
		// use IasClient stub unless test IAS ENV vars exist
		log.Println("Using mock IAS client as TEST_EVENTING_AUTH_IAS_URL, TEST_EVENTING_AUTH_IAS_USER, and TEST_EVENTING_AUTH_IAS_PASSWORD are missing")
		originalNewIasClientFunc = ias.NewIasClient
		mockNewIasClientFunc := func(iasTenantUrl, user, password string) (ias.Client, error) {
			return iasClientStub{}, nil
		}
		ias.NewIasClient = mockNewIasClientFunc
	}

	reconciler := controllers.NewEventingAuthReconciler(mgr.GetClient(), mgr.GetScheme(), time.Second*1, time.Second*3)

	Expect(reconciler.SetupWithManager(mgr)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).Should(Succeed())
	}()

}, NodeTimeout(60*time.Second))

var _ = AfterSuite(func() {
	cancel()
	ias.ReadCredentials = originalReadCredentialsFunc
	if !existIasCreds() {
		ias.NewIasClient = originalNewIasClientFunc
	}
	By("Tearing down the test environment")
	stopTestEnv(testEnv)
	stopTestEnv(targetClusterTestEnv)
})

func stopTestEnv(env *envtest.Environment) {
	err := env.Stop()

	// Suggested workaround for timeout issue https://github.com/kubernetes-sigs/controller-runtime/issues/1571#issuecomment-1005575071
	if err != nil {
		time.Sleep(3 * time.Second)
	}
	Expect(err).NotTo(HaveOccurred())
}

func existIasCreds() bool {
	return iasUrl != "" && iasUsername != "" && iasPassword != ""
}

type iasClientStub struct {
}

func (i iasClientStub) CreateApplication(_ context.Context, name string) (ias.Application, error) {
	return ias.NewApplication(fmt.Sprintf("id-for-%s", name), fmt.Sprintf("client-id-for-%s", name), "test-client-secret", "https://test-token-url.com/token"), nil
}

func (i iasClientStub) DeleteApplication(_ context.Context, _ string) error {
	return nil
}

func (i iasClientStub) GetCredentials() *ias.Credentials {
	return &ias.Credentials{}
}

func initTargetClusterConfig() (client.Client, error) {
	targetK8sCfgPath := os.Getenv("TEST_EVENTING_AUTH_TARGET_KUBECONFIG_PATH")

	var err error
	var clientConfig *rest.Config

	if targetK8sCfgPath == "" {
		log.Println("Starting local KubeAPI and ETCD server as target TEST_EVENTING_AUTH_TARGET_KUBECONFIG_PATH is missing")
		targetClusterTestEnv = &envtest.Environment{}

		clientConfig, err = targetClusterTestEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(clientConfig).NotTo(BeNil())

		// We create a new user since this is the easiest way to get the kubeconfig in the right format to store it in the secret.
		adminUser, err := targetClusterTestEnv.ControlPlane.AddUser(envtest.User{
			Name:   "envtest-admin",
			Groups: []string{"system:masters"},
		}, nil)
		Expect(err).NotTo(HaveOccurred())

		kubeconfig, err := adminUser.KubeConfig()
		Expect(err).NotTo(HaveOccurred())
		targetClusterK8sCfg = base64.StdEncoding.EncodeToString(kubeconfig)
	} else {
		log.Println("Using a K8s cluster with kubeconfig file: " + targetK8sCfgPath)
		kubeconfig, err := os.ReadFile(targetK8sCfgPath)
		Expect(err).NotTo(HaveOccurred())
		targetClusterK8sCfg = base64.StdEncoding.EncodeToString(kubeconfig)

		config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
		Expect(err).NotTo(HaveOccurred())

		targetClusterTestEnv = &envtest.Environment{
			Config:             config,
			UseExistingCluster: pointer.Bool(true),
		}

		clientConfig, err = targetClusterTestEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(clientConfig).NotTo(BeNil())
	}

	return client.New(clientConfig, client.Options{})
}
