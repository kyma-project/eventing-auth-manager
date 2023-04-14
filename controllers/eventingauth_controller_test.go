package controllers_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EventingAuth Controller", func() {

	Context("When creating EventingAuth CR", func() {

		It("should create IAS app and secret", func() {
			cr := v1alpha1.EventingAuth{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: namespace,
				},
			}

			By("Creating EventingAuth CR")
			Expect(k8sClient.Create(context.TODO(), &cr)).Should(Succeed())

			verifyEventingAuthState("test", v1alpha1.StateOk)
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
