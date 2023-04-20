package controllers_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var appSecretObjectKey = ctrlClient.ObjectKey{Name: "eventing-auth-application", Namespace: "kyma-system"}

// Since all tests use the same target cluster and therefore share the same application secret, they need to be executed serially
var _ = Describe("EventingAuth Controller", Serial, func() {

	BeforeEach(func() {
		// We need to clean up the secret before each test, because a failing test could leave the secret in the cluster and this can lead to cascading failures.
		secret := corev1.Secret{}
		err := targetClusterK8sClient.Get(context.TODO(), appSecretObjectKey, &secret)
		if err != nil {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		} else {
			Expect(targetClusterK8sClient.Delete(context.TODO(), &secret)).Should(Succeed())
		}

	})
	//TODO: Change tests to use table tests
	Context("When creating EventingAuth CR", func() {

		It("should create secret with IAS applications credentials", func() {
			cr := createEventingAuth()
			s := createKubeconfigSecret(cr)
			defer func() {
				teardownEventingAuth(cr)
				deleteKubeconfigSecret(s)
			}()
			verifyEventingAuthState(cr, v1alpha1.StateOk)
			verifySecretExistsOnTargetCluster()
		})
	})

	Context("When deleting EventingAuth CR", func() {

		It("should delete secret with IAS applications credentials", func() {
			cr := createEventingAuth()
			s := createKubeconfigSecret(cr)
			defer func() {
				teardownEventingAuth(cr)
				deleteKubeconfigSecret(s)
			}()
			verifyEventingAuthState(cr, v1alpha1.StateOk)
			verifySecretExistsOnTargetCluster()
			deleteEventingAuth(cr)
			verifySecretDoesNotExistOnTargetCluster(s)
		})
	})

	It("should create new secret on target cluster on time-based reconciliation when secret was deleted manually", func() {
		cr := createEventingAuth()
		s := createKubeconfigSecret(cr)
		defer func() {
			teardownEventingAuth(cr)
			deleteKubeconfigSecret(s)
		}()
		verifyEventingAuthState(cr, v1alpha1.StateOk)
		secret := verifySecretExistsOnTargetCluster()
		deleteSecretOnTargetCluster(secret)
		verifySecretDoesNotExistOnTargetCluster(secret)
		verifySecretExistsOnTargetCluster()
	})
})

func deleteSecretOnTargetCluster(secret *corev1.Secret) {
	By("Deleting secret on target cluster")
	Expect(targetClusterK8sClient.Delete(context.TODO(), secret)).Should(Succeed())
}

func createEventingAuth() *v1alpha1.EventingAuth {
	e := v1alpha1.EventingAuth{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateCrName(),
			Namespace: namespace,
		},
	}

	By("Creating EventingAuth CR")
	Expect(k8sClient.Create(context.TODO(), &e)).Should(Succeed())

	return &e
}

func verifySecretDoesNotExistOnTargetCluster(s *corev1.Secret) {
	By("Verifying that IAS application secret does not exist on target cluster")
	Eventually(func(g Gomega) {
		err := targetClusterK8sClient.Get(context.TODO(), appSecretObjectKey, s)
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}, defaultTimeout).Should(Succeed())
}

func verifySecretExistsOnTargetCluster() *corev1.Secret {
	By("Verifying that IAS application secret exists on target cluster")
	s := corev1.Secret{}
	Eventually(func(g Gomega) {
		g.Expect(targetClusterK8sClient.Get(context.TODO(), appSecretObjectKey, &s)).Should(Succeed())
	}, defaultTimeout).Should(Succeed())

	return &s
}

func deleteKubeconfigSecret(s *corev1.Secret) {
	By(fmt.Sprintf("Deleting secret %s as part of teardown", s.Name))
	Expect(k8sClient.Delete(context.TODO(), s)).Should(Succeed())
	Eventually(func(g Gomega) {
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(s), &corev1.Secret{})
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}, defaultTimeout).Should(Succeed())
}

func generateCrName() string {
	return fmt.Sprintf(uuid.New().String())
}

func verifyEventingAuthState(cr *v1alpha1.EventingAuth, state v1alpha1.State) {
	By(fmt.Sprintf("Verifying that EventingAuth %s has status %s", cr.Name, state))
	Eventually(func(g Gomega) {
		e := v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(cr), &e)).Should(Succeed())
		g.Expect(e.Status.State).NotTo(BeNil())
		g.Expect(e.Status.State).To(Equal(state))
	}, defaultTimeout).Should(Succeed())
}

func deleteEventingAuth(e *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Deleting EventingAuth %s", e.Name))
	Expect(k8sClient.Delete(context.TODO(), e)).Should(Succeed())
	Eventually(func(g Gomega) {
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(e), &v1alpha1.EventingAuth{})
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}, defaultTimeout).Should(Succeed())
}

func teardownEventingAuth(e *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Deleting EventingAuth %s as part of teardown", e.Name))
	if err := k8sClient.Delete(context.TODO(), e); err != nil && !errors.IsNotFound(err) {
		Expect(err).Should(Succeed())
	}
}

func createKubeconfigSecret(e *v1alpha1.EventingAuth) *corev1.Secret {
	By("Creating secret with kubeconfig of target cluster")
	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubeconfig-%s", e.Name),
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			"config": []byte(targetClusterK8sCfg),
		},
	}
	Expect(k8sClient.Create(context.TODO(), &kubeconfigSecret)).Should(Succeed())
	return &kubeconfigSecret
}
