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
	//+kubebuilder:scaffold:imports
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	operatorv1alpha1 "github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-auth-manager/controllers"
	"github.com/kyma-project/eventing-auth-manager/internal/skr"
	kymav1beta1 "github.com/kyma-project/lifecycle-manager/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	defaultTimeout = time.Second * 60
)

var (
	cfg                    *rest.Config
	k8sClient              client.Client
	ctx                    context.Context
	cancel                 context.CancelFunc
	targetClusterK8sCfg    string
	targetClusterK8sClient client.Client
	testEnv                *envtest.Environment
	targetClusterTestEnv   *envtest.Environment
	iasUrl                 string
	iasUsername            string
	iasPassword            string
	useExistingCluster     bool
	kcpNs                  *corev1.Namespace
	kymaNs                 *corev1.Namespace
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
	useExistingCluster = os.Getenv("USE_EXISTING_CLUSTER") == "true"

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases"), filepath.Join("..", "config", "crd", "external")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &useExistingCluster,
	}

	var err error

	// Start Target Cluster
	targetClusterK8sClient, err = initTargetClusterConfig()
	Expect(err).NotTo(HaveOccurred())

	// In case we are using an existing cluster the namespace of the application secret might already exist, so we need to guard against that.
	kymaNs = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: skr.ApplicationSecretNamespace}}
	err = targetClusterK8sClient.Get(context.TODO(), client.ObjectKeyFromObject(kymaNs), &corev1.Namespace{})
	if err != nil {
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
		err := targetClusterK8sClient.Create(context.TODO(), kymaNs)
		Expect(err).NotTo(HaveOccurred())
	}

	// Start the test cluster
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	Expect(kymav1beta1.AddToScheme(scheme.Scheme)).Should(Succeed())
	Expect(operatorv1alpha1.AddToScheme(scheme.Scheme)).Should(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	kcpNs = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: skr.KcpNamespace},
		Spec:       corev1.NamespaceSpec{},
	}
	Expect(k8sClient.Create(context.TODO(), kcpNs)).Should(Succeed())

	if existIasCreds() {
		createIasCredsSecret(iasUrl, iasUsername, iasPassword)
	}

	testSyncPeriod := time.Second * 1
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		Cache: cache.Options{SyncPeriod: &testSyncPeriod},
	})
	Expect(err).NotTo(HaveOccurred())

	// Since we are replacing in some test scenarios the original functions we need to keep them, so we are able to reset them after the tests.
	storeOriginalsOfStubbedFunctions()

	kymaReconciler := controllers.NewKymaReconciler(mgr.GetClient(), mgr.GetScheme())
	Expect(kymaReconciler.SetupWithManager(mgr)).Should(Succeed())

	eventingAuthReconciler := controllers.NewEventingAuthReconciler(mgr.GetClient(), mgr.GetScheme())
	Expect(eventingAuthReconciler.SetupWithManager(mgr)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).Should(Succeed())
	}()

}, NodeTimeout(60*time.Second))

var _ = AfterSuite(func() {
	cancel()
	revertSkrNewClientStub()
	revertReadCredentialsStub()
	if !existIasCreds() {
		revertIasNewClientStub()
	} else {
		deleteIasCredsSecret()
	}
	By("Cleaning up the kyma-system namespace in SKR cluster")
	Expect(targetClusterK8sClient.Delete(context.TODO(), kymaNs)).Should(Succeed())
	By("Cleaning up the kcp-system namespace in kcp cluster")
	Expect(k8sClient.Delete(context.TODO(), kcpNs)).Should(Succeed())

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
		targetClusterK8sCfg = string(kubeconfig)
	} else {
		log.Println("Using a K8s cluster with kubeconfig file: " + targetK8sCfgPath)
		kubeconfig, err := os.ReadFile(targetK8sCfgPath)
		Expect(err).NotTo(HaveOccurred())
		targetClusterK8sCfg = string(kubeconfig)

		config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
		Expect(err).NotTo(HaveOccurred())

		targetClusterTestEnv = &envtest.Environment{
			Config:             config,
			UseExistingCluster: ptr.To(true),
		}

		clientConfig, err = targetClusterTestEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(clientConfig).NotTo(BeNil())
	}

	return client.New(clientConfig, client.Options{})
}
