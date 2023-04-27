package controllers_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var appSecretObjectKey = ctrlClient.ObjectKey{Name: "eventing-webhook-auth", Namespace: "kyma-system"}

// Since all tests use the same target cluster and therefore share the same application secret, they need to be executed serially
var _ = Describe("EventingAuth Controller", Serial, func() {

	BeforeEach(func() {
		deleteEventingAuths()
		// We also need to clean up the app secret before each test, because a failing test could leave the secret in the cluster and this can lead to cascading failures.
		deleteApplicationSecretOnTargetCluster()
		deleteKubeconfigSecrets()
	})

	//TODO: Change tests to use table tests
	Context("Creating EventingAuth CR", func() {

		It("should create secret with IAS applications credentials", func() {
			cr := createEventingAuthAndKubeconfigSecret()
			verifyEventingAuthStatus(cr, v1alpha1.StateReady)
			verifySecretExistsOnTargetCluster()
		})

		It("should have CR status NotReady when IAS app creation fails", func() {
			// TODO: Implement test
		})

		It("should have CR status NotReady when secret creation on target cluster fails", func() {
			// TODO: Implement test
		})
	})

	Context("Deleting EventingAuth CR", func() {

		It("should delete secret with IAS applications credentials", func() {
			cr := createEventingAuthAndKubeconfigSecret()
			verifyEventingAuthStatus(cr, v1alpha1.StateReady)
			s := verifySecretExistsOnTargetCluster()
			deleteEventingAuth(cr)
			verifySecretDoesNotExistOnTargetCluster(s)
		})
	})

	Context("Time-based EventingAuth CR reconciliation", func() {

		It("should create new secret on target cluster on next reconciliation when secret was deleted manually", func() {
			cr := createEventingAuthAndKubeconfigSecret()
			verifyEventingAuthStatus(cr, v1alpha1.StateReady)
			secret := verifySecretExistsOnTargetCluster()
			deleteSecretOnTargetCluster(secret)
			verifySecretDoesNotExistOnTargetCluster(secret)
			verifySecretExistsOnTargetCluster()
		})

	})

	Context("Error during reconciliation", func() {

		It("should retry and create application when first attempt of application creation failed", func() {
			// TODO: Implement test
		})

		It("should retry and create secret when first attempt of secret creation failed", func() {
			// TODO: Implement test
		})
	})

})

func createEventingAuthAndKubeconfigSecret() *v1alpha1.EventingAuth {
	crName := generateCrName()
	createKubeconfigSecret(crName)
	cr := createEventingAuth(crName)
	return cr
}

func deleteSecretOnTargetCluster(secret *corev1.Secret) {
	By("Deleting secret on target cluster")
	Expect(targetClusterK8sClient.Delete(context.TODO(), secret)).Should(Succeed())
}

func createEventingAuth(name string) *v1alpha1.EventingAuth {
	e := v1alpha1.EventingAuth{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
		g.Expect(s.Data).To(HaveKey("client_id"))
		g.Expect(s.Data["client_id"]).ToNot(BeEmpty())
		g.Expect(s.Data).To(HaveKey("client_secret"))
		g.Expect(s.Data["client_secret"]).ToNot(BeEmpty())
		g.Expect(s.Data).To(HaveKey("token_url"))
		g.Expect(s.Data["token_url"]).To(ContainSubstring("/token"))
	}, defaultTimeout).Should(Succeed())

	return &s
}

func generateCrName() string {
	return uuid.New().String()
}

func verifyEventingAuthStatus(cr *v1alpha1.EventingAuth, state v1alpha1.State) {
	By(fmt.Sprintf("Verifying that EventingAuth %s has status %s", cr.Name, state))
	Eventually(func(g Gomega) {
		e := v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(cr), &e)).Should(Succeed())
		g.Expect(e.Status.State).NotTo(BeNil())
		g.Expect(e.Status.State).To(Equal(state))

		g.Expect(e.Status.Conditions).To(ContainElements(
			conditionMatcher(
				string(v1alpha1.ConditionApplicationReady),
				metav1.ConditionTrue,
				v1alpha1.ConditionReasonApplicationCreated,
				v1alpha1.ConditionMessageApplicationCreated),
			conditionMatcher(
				string(v1alpha1.ConditionSecretReady),
				metav1.ConditionTrue,
				v1alpha1.ConditionReasonSecretCreated,
				v1alpha1.ConditionMessageSecretCreated),
		))
	}, defaultTimeout).Should(Succeed())
}

func conditionMatcher(t string, s metav1.ConditionStatus, r, m string) types.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(t),
		"Status":  Equal(s),
		"Reason":  Equal(r),
		"Message": Equal(m),
	})
}
func deleteEventingAuth(e *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Deleting EventingAuth %s", e.Name))
	Expect(k8sClient.Delete(context.TODO(), e)).Should(Succeed())
	Eventually(func(g Gomega) {
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(e), &v1alpha1.EventingAuth{})
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}, defaultTimeout).Should(Succeed())
}

func deleteEventingAuths() {
	eaList := v1alpha1.EventingAuthList{}
	Expect(k8sClient.List(context.TODO(), &eaList, ctrlClient.InNamespace(namespace))).Should(Succeed())

	for _, ea := range eaList.Items {
		if err := k8sClient.Delete(context.TODO(), &ea); err != nil && !errors.IsNotFound(err) {
			Expect(err).Should(Succeed())
		}
	}
}

func deleteKubeconfigSecrets() {
	sList := corev1.SecretList{}
	Expect(k8sClient.List(context.TODO(), &sList, ctrlClient.InNamespace("kcp-system"))).Should(Succeed())

	for _, s := range sList.Items {
		if err := k8sClient.Delete(context.TODO(), &s); err != nil && !errors.IsNotFound(err) {
			Expect(err).Should(Succeed())
		}
	}
}

func deleteApplicationSecretOnTargetCluster() {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appSecretObjectKey.Name,
			Namespace: appSecretObjectKey.Namespace,
		},
	}

	if err := targetClusterK8sClient.Delete(context.TODO(), &secret); err != nil {
		Expect(errors.IsNotFound(err)).To(BeTrue())
	}
}

func createKubeconfigSecret(crName string) *corev1.Secret {
	By("Creating secret with kubeconfig of target cluster")
	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubeconfig-%s", crName),
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			"config": []byte(targetClusterK8sCfg),
		},
	}
	Expect(k8sClient.Create(context.TODO(), &kubeconfigSecret)).Should(Succeed())
	return &kubeconfigSecret
}
