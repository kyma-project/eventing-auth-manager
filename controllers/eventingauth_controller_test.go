package controllers_test

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-auth-manager/internal/skr"
	gtypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var appSecretObjectKey = ctrlClient.ObjectKey{Name: skr.ApplicationSecretName, Namespace: skr.ApplicationSecretNamespace}

// Since all tests use the same target cluster and therefore share the same application secret, they need to be executed serially.
var _ = Describe("EventingAuth Controller happy tests", Serial, Ordered, func() {
	BeforeAll(func() {
		// We allow the happy path tests to use the real IAS client if the test env vars are set
		if !existIasCreds() {
			// use IasClient stub unless test IAS ENV vars exist
			log.Println("Using mock IAS client as TEST_EVENTING_AUTH_IAS_URL, TEST_EVENTING_AUTH_IAS_USER, and TEST_EVENTING_AUTH_IAS_PASSWORD are missing")
			stubSuccessfulIasAppCreation()
		}
	})

	AfterAll(func() {
		revertIasNewClientStub()
	})

	Context("Creating and Deleting EventingAuth CR", func() {
		var (
			eventingAuth *v1alpha1.EventingAuth
			crName       string
		)
		BeforeEach(func() {
			crName = generateCrName()
			createKubeconfigSecret(crName)
		})

		AfterEach(func() {
			deleteEventingAuthAndVerify(eventingAuth)
			deleteKubeconfigSecret(crName)
		})

		It("should delete secret with IAS applications credentials", func() {
			eventingAuth = createEventingAuth(crName)
			verifyEventingAuthStatusReady(eventingAuth)

			// Time-based EventingAuth CR reconciliation
			secret := verifySecretExistsOnTargetCluster()
			deleteSecretOnTargetCluster(secret)
			verifySecretDoesNotExistOnTargetCluster()
			verifySecretExistsOnTargetCluster()

			// Testing deletion
			deleteEventingAuthAndVerify(eventingAuth)
			verifySecretDoesNotExistOnTargetCluster()
		})
		It("should deletion succeed when kubeconfig secret is missing", func() {
			// given
			eventingAuth = createEventingAuth(crName)
			verifyEventingAuthStatusReady(eventingAuth)
			// when
			deleteKubeconfigSecret(crName)
			// then, eventingAuth deletion should succeed
			deleteEventingAuthAndVerify(eventingAuth)

			// clean-up by recreating kubeconfig secret
			createKubeconfigSecret(crName)
			secret := verifySecretExistsOnTargetCluster()
			deleteSecretOnTargetCluster(secret)
		})
	})
})

var _ = Describe("EventingAuth Controller unhappy tests", Serial, Ordered, func() {
	var (
		eventingAuth *v1alpha1.EventingAuth
		crName       string
	)

	BeforeEach(func() {
		crName = generateCrName()
		createKubeconfigSecret(crName)
	})

	AfterEach(func() {
		// as these tests uses mock IAS we can force delete EventingAuth CR
		deleteEventingAuthAndVerify(eventingAuth)
		deleteApplicationSecretOnTargetCluster()
		deleteKubeconfigSecret(crName)
		// set SKR client to original
		revertReadCredentialsStub()
		revertSkrNewClientStub()
		revertIasNewClientStub()
	})

	It("should have CR status NotReady when IAS app creation fails", func() {
		stubFailedIasAppCreation()
		eventingAuth = createEventingAuth(crName)
		verifyEventingAuthStatusNotReadyAppCreationFailed(eventingAuth)
	})

	It("should have CR status NotReady when secret creation on target cluster fails", func() {
		stubSuccessfulIasAppCreation()
		stubFailedSkrSecretCreation()
		eventingAuth = createEventingAuth(crName)
		verifyEventingAuthStatusNotReadySecretCreationFailed(eventingAuth)
	})

	It("should retry and create application when first attempt of application creation failed", func() {
		stubFailedIasAppCreation()
		stubSuccessfulSkrSecretCreation()
		eventingAuth = createEventingAuth(crName)
		verifyEventingAuthStatusNotReadyAppCreationFailed(eventingAuth)
		stubSuccessfulIasAppCreation()
		verifyEventingAuthStatusReady(eventingAuth)
	})

	It("should retry and create secret when first attempt of secret creation failed", func() {
		stubSuccessfulIasAppCreation()
		stubFailedSkrSecretCreation()
		eventingAuth = createEventingAuth(crName)
		verifyEventingAuthStatusNotReadySecretCreationFailed(eventingAuth)
		stubSuccessfulSkrSecretCreation()
		verifyEventingAuthStatusReady(eventingAuth)
	})
})

func deleteSecretOnTargetCluster(secret *corev1.Secret) {
	By("Deleting secret on target cluster")
	Expect(targetClusterK8sClient.Delete(context.TODO(), secret)).Should(Succeed())
}

func createEventingAuth(name string) *v1alpha1.EventingAuth {
	e := v1alpha1.EventingAuth{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: skr.KcpNamespace,
		},
	}

	By("Creating EventingAuth CR")
	Expect(k8sClient.Create(context.TODO(), &e)).Should(Succeed())

	return &e
}

func verifySecretDoesNotExistOnTargetCluster() {
	By("Verifying that IAS application secret does not exist on target cluster")
	Eventually(func(g Gomega) {
		err := targetClusterK8sClient.Get(context.TODO(), appSecretObjectKey, &corev1.Secret{})
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
		g.Expect(s.Data).To(HaveKey("certs_url"))
		g.Expect(s.Data["certs_url"]).To(ContainSubstring("/certs"))
	}, defaultTimeout).Should(Succeed())

	return &s
}

func generateCrName() string {
	return uuid.New().String()
}

func verifyEventingAuthStatusReady(cr *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Verifying that EventingAuth %s has status %s", cr.Name, v1alpha1.StateReady))
	Eventually(func(g Gomega) {
		e := v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(cr), &e)).Should(Succeed())
		g.Expect(e.Status.State).NotTo(BeNil())
		g.Expect(e.Status.State).To(Equal(v1alpha1.StateReady))

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

func verifyEventingAuthStatusNotReadyAppCreationFailed(cr *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Verifying that EventingAuth %s has status %s", cr.Name, v1alpha1.StateNotReady))
	Eventually(func(g Gomega) {
		e := v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(cr), &e)).Should(Succeed())
		g.Expect(e.Status.State).NotTo(BeNil())
		g.Expect(e.Status.State).To(Equal(v1alpha1.StateNotReady))

		g.Expect(e.Status.Conditions).To(ContainElements(
			conditionMatcher(
				string(v1alpha1.ConditionApplicationReady),
				metav1.ConditionFalse,
				v1alpha1.ConditionReasonApplicationCreationFailed,
				"stubbed IAS application creation error"),
		))
	}, defaultTimeout).Should(Succeed())
}

func verifyEventingAuthStatusNotReadySecretCreationFailed(cr *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Verifying that EventingAuth %s has status %s", cr.Name, v1alpha1.StateNotReady))
	Eventually(func(g Gomega) {
		e := v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(cr), &e)).Should(Succeed())
		g.Expect(e.Status.State).NotTo(BeNil())
		g.Expect(e.Status.State).To(Equal(v1alpha1.StateNotReady))

		g.Expect(e.Status.Conditions).To(ContainElements(
			conditionMatcher(
				string(v1alpha1.ConditionSecretReady),
				metav1.ConditionFalse,
				v1alpha1.ConditionReasonSecretCreationFailed,
				"stubbed skr secret creation error"),
		))
	}, defaultTimeout).Should(Succeed())
}

func conditionMatcher(t string, s metav1.ConditionStatus, r, m string) gtypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(t),
		"Status":  Equal(s),
		"Reason":  Equal(r),
		"Message": Equal(m),
	})
}
func deleteEventingAuthAndVerify(e *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Deleting EventingAuth %s", e.Name))
	if err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(e), &v1alpha1.EventingAuth{}); err != nil {
		Expect(errors.IsNotFound(err)).Should(BeTrue())
		return
	}
	Expect(k8sClient.Delete(context.TODO(), e)).Should(Succeed())
	Eventually(func(g Gomega) {
		latestEAuth := &v1alpha1.EventingAuth{}
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(e), latestEAuth)
		g.Expect(err).Should(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).Should(BeTrue())
	}, defaultTimeout).Should(Succeed())
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

func createKubeconfigSecret(crName string) {
	By("Creating secret with kubeconfig of target cluster")
	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubeconfig-%s", crName),
			Namespace: skr.KcpNamespace,
		},
		Data: map[string][]byte{
			"config": []byte(targetClusterK8sCfg),
		},
	}
	Expect(k8sClient.Create(context.TODO(), &kubeconfigSecret)).Should(Succeed())
}

func deleteKubeconfigSecret(crName string) {
	By("Deleting secret with kubeconfig of target cluster")
	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubeconfig-%s", crName),
			Namespace: skr.KcpNamespace,
		},
	}
	Eventually(func(g Gomega) {
		err := k8sClient.Delete(context.TODO(), &kubeconfigSecret)
		if err != nil {
			g.Expect(errors.IsNotFound(err)).Should(BeTrue())
		}
	}, defaultTimeout).Should(Succeed())
}
