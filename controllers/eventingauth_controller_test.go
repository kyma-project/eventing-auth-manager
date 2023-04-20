package controllers_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EventingAuth Controller", func() {

	//TODO: Change tests to use table tests
	Context("When creating EventingAuth CR", func() {

		It("should create secret with IAS applications credentials", func() {
			cr := v1alpha1.EventingAuth{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: namespace,
				},
			}

			By("Creating EventingAuth CR")
			Expect(k8sClient.Create(context.TODO(), &cr)).Should(Succeed())

			verifyEventingAuthState("test", v1alpha1.StateReady)

			By("Verifying that IAS application secret exists")
			secret := corev1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: "eventing-auth-application", Namespace: "kyma-system"}, &secret)).Should(Succeed())
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
})

func verifyEventingAuthState(resourceName string, state v1alpha1.State) {
	By(fmt.Sprintf("Verifying that EventingAuth %s has status %s", resourceName, state))
	Eventually(func(g Gomega) {
		e := v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: resourceName, Namespace: namespace}, &e)).Should(Succeed())
		g.Expect(e.Status.State).NotTo(BeNil())
		g.Expect(e.Status.State).To(Equal(state))
	}, defaultTimeout).Should(Succeed())
}
