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

const applicationSecretName = "eventing-auth-application"
const applicationSecretNamespace = "kyma-system"

// Since all tests use the same target cluster and therefore share the same application secret, they need to be executed serially
var _ = Describe("EventingAuth Controller", Serial, func() {

	BeforeEach(func() {
		// We need to clean up the secret before each test, because a failing test could leave the secret in the cluster and this can lead to cascading failures.
		secret := corev1.Secret{}
		err := targetClusterK8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: applicationSecretName, Namespace: applicationSecretName}, &secret)
		if err != nil {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		} else {
			Expect(targetClusterK8sClient.Delete(context.TODO(), &secret)).Should(Succeed())
		}

	})
	//TODO: Change tests to use table tests
	Context("When creating EventingAuth CR", func() {

		It("should create secret with IAS applications credentials", func() {
			cr := v1alpha1.EventingAuth{
				ObjectMeta: metav1.ObjectMeta{
					Name:      generateCrName(),
					Namespace: namespace,
				},
			}

			By("Creating secret with kubeconfig of target cluster")
			k8sCfgSecret := createKubeconfigSecret(cr.Name)
			defer func() {
				deleteKubeconfigSecret(k8sCfgSecret)
			}()

			By("Creating EventingAuth CR")
			Expect(k8sClient.Create(context.TODO(), &cr)).Should(Succeed())
			defer func() {
				deleteEventingAuth(&cr)
			}()

			verifyEventingAuthState(&cr, v1alpha1.StateReady)

			By("Verifying that IAS application secret exists on target cluster")
			secret := corev1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(targetClusterK8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: applicationSecretName, Namespace: applicationSecretNamespace}, &secret)).Should(Succeed())
				g.Expect(secret.Data).To(HaveKey("client_id"))
				g.Expect(secret.Data["client_id"]).ToNot(BeEmpty())
				g.Expect(secret.Data).To(HaveKey("client_secret"))
				g.Expect(secret.Data["client_secret"]).ToNot(BeEmpty())
				g.Expect(secret.Data).To(HaveKey("token_url"))
				expectedPartlyTokenUrl := fmt.Sprintf("/oauth2/token?grant_type=client_credentials&client_id=%s", secret.Data["client_id"])
				g.Expect(secret.Data["token_url"]).To(ContainSubstring(expectedPartlyTokenUrl))
			}, defaultTimeout).Should(Succeed())
		})
	})

	Context("When deleting EventingAuth CR", func() {

		It("should delete secret with IAS applications credentials", func() {
			cr := v1alpha1.EventingAuth{
				ObjectMeta: metav1.ObjectMeta{
					Name:      generateCrName(),
					Namespace: namespace,
				},
			}

			By("Creating secret with kubeconfig of target cluster")
			k8sCfgSecret := createKubeconfigSecret(cr.Name)
			defer func() {
				deleteKubeconfigSecret(k8sCfgSecret)
			}()

			By("Creating EventingAuth CR")
			Expect(k8sClient.Create(context.TODO(), &cr)).Should(Succeed())

			verifyEventingAuthState(&cr, v1alpha1.StateOk)

			By("Verifying that IAS application secret exists on target cluster")
			secret := corev1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(targetClusterK8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: applicationSecretName, Namespace: applicationSecretNamespace}, &secret)).Should(Succeed())
			}, defaultTimeout).Should(Succeed())

			By("Deleting EventingAuth CR")
			Expect(k8sClient.Delete(context.TODO(), &cr)).Should(Succeed())

			By("Verifying that IAS application secret does not exist on target cluster")
			Eventually(func(g Gomega) {
				err := targetClusterK8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: applicationSecretName, Namespace: applicationSecretNamespace}, &secret)
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			}, defaultTimeout).Should(Succeed())
		})
	})
})

func deleteKubeconfigSecret(secret *corev1.Secret) {
	By(fmt.Sprintf("Deleting secret %s as part of teardown", secret.Name))
	Expect(k8sClient.Delete(context.TODO(), secret)).Should(Succeed())
	Eventually(func(g Gomega) {
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(secret), &corev1.Secret{})
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

func deleteEventingAuth(eventingAuth *v1alpha1.EventingAuth) {
	By(fmt.Sprintf("Deleting EventingAuth %s as part of teardown", eventingAuth.Name))
	Expect(k8sClient.Delete(context.TODO(), eventingAuth)).Should(Succeed())
	Eventually(func(g Gomega) {
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(eventingAuth), &v1alpha1.EventingAuth{})
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}, defaultTimeout).Should(Succeed())
}

func createKubeconfigSecret(targetClusterId string) *corev1.Secret {
	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubeconfig-%s", targetClusterId),
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			"config": []byte(targetClusterK8sCfg),
		},
	}
	Expect(k8sClient.Create(context.TODO(), &kubeconfigSecret)).Should(Succeed())
	return &kubeconfigSecret
}
