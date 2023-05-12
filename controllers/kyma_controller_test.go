package controllers_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	kymav1beta1 "github.com/kyma-project/lifecycle-manager/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Since all tests use the same target cluster and therefore share the same application secret, they need to be executed serially
var _ = Describe("Kyma Controller", Serial, func() {
	AfterEach(func() {
		deleteKymaResources()
		deleteKubeconfigSecrets()
	})

	Context("Creating Kyma CR", func() {
		It("should create Kyma CR", func() {
			crName := generateCrName()
			createKubeconfigSecret(crName)
			kyma := createKymaResource(crName)
			verifyEventingAuth(kyma.Namespace, kyma.Name)
		})
	})

	Context("Creating and Deleting Kyma CR", func() {
		It("should delete Kyma CR", func() {
			if !useExistingCluster {
				Skip("Skipping for a not real cluster as k8s garbage collection works with a real cluster")
			}
			// create Kyma CR
			crName := generateCrName()
			createKubeconfigSecret(crName)
			kyma := createKymaResource(crName)
			verifyEventingAuth(kyma.Namespace, kyma.Name)
			// delete Kyma CR
			deleteKymaResource(kyma)
			By(fmt.Sprintf("Verifying whether EventingAuth CR is deleted %s", kyma.Namespace+"/"+kyma.Name))
			Eventually(func(g Gomega) {
				eventingAuth := &v1alpha1.EventingAuth{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: kyma.Namespace, Name: kyma.Name}, eventingAuth)
				g.Expect(err).ShouldNot(BeNil())
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			}, defaultTimeout).Should(Succeed())
		})
	})
})

func verifyEventingAuth(namespace, name string) {
	nsName := types.NamespacedName{Namespace: namespace, Name: name}
	By(fmt.Sprintf("Verifying Kyma CR %s", nsName.String()))
	Eventually(func(g Gomega) {
		eventingAuth := &v1alpha1.EventingAuth{}
		g.Expect(k8sClient.Get(context.TODO(), nsName, eventingAuth)).Should(Succeed())
		g.Expect(eventingAuth.Name).To(Equal(name))
		g.Expect(eventingAuth.Namespace).To(Equal(namespace))
	}, defaultTimeout).Should(Succeed())
}

func createKymaResource(name string) *kymav1beta1.Kyma {
	kyma := kymav1beta1.Kyma{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kymav1beta1.KymaSpec{
			Modules: []kymav1beta1.Module{kymav1beta1.Module{Name: "nats"}},
			Channel: "alpha",
		},
	}

	By("Creating Kyma CR")
	Expect(k8sClient.Create(context.TODO(), &kyma)).Should(Succeed())

	return &kyma
}

func deleteKymaResource(kyma *kymav1beta1.Kyma) {
	By(fmt.Sprintf("Deleting Kyma %s", kyma.Name))
	Expect(k8sClient.Delete(context.TODO(), kyma)).Should(Succeed())
	Eventually(func(g Gomega) {
		err := k8sClient.Get(context.TODO(), ctrlClient.ObjectKeyFromObject(kyma), &kymav1beta1.Kyma{})
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}, defaultTimeout).Should(Succeed())
}

func deleteKymaResources() {
	kymaList := kymav1beta1.KymaList{}
	Expect(k8sClient.List(context.TODO(), &kymaList, ctrlClient.InNamespace(namespace))).Should(Succeed())

	for _, kyma := range kymaList.Items {
		if err := k8sClient.Delete(context.TODO(), &kyma); err != nil && !errors.IsNotFound(err) {
			Expect(err).Should(Succeed())
		}
	}
}
