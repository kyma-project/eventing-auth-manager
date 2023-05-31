package controllers_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/eventing-auth-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-auth-manager/controllers"
	"github.com/kyma-project/eventing-auth-manager/internal/skr"
	kymav1beta1 "github.com/kyma-project/lifecycle-manager/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Since all tests use the same target cluster and therefore share the same application secret, they need to be executed serially
var _ = Describe("Kyma Controller", Serial, func() {
	var kyma *kymav1beta1.Kyma
	var crName string
	BeforeEach(func() {
		crName = generateCrName()
		createKubeconfigSecret(crName)
	})

	AfterEach(func() {
		deleteKubeconfigSecrets()
	})

	Context("Creating Kyma CR", func() {
		It("should create Kyma CR", func() {
			kyma = createKymaResource(crName)

			verifyEventingAuth(kyma.Namespace, kyma.Name)

			deleteKymaResource(kyma)
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
		g.Expect(eventingAuth.Status.State).To(Equal(v1alpha1.StateReady))
		g.Expect(eventingAuth.Status.Application).ShouldNot(BeNil())
		g.Expect(eventingAuth.Status.Application.Name).To(Equal(name))
		g.Expect(eventingAuth.Status.Application.UUID).ShouldNot(BeEmpty())
		g.Expect(eventingAuth.Status.AuthSecret).ShouldNot(BeNil())
		g.Expect(eventingAuth.Status.AuthSecret.NamespacedName).To(Equal((skr.ApplicationSecretNamespace + "/" + skr.ApplicationSecretName)))
		g.Expect(eventingAuth.Status.AuthSecret.ClusterId).ShouldNot(BeEmpty())
	}, defaultTimeout).Should(Succeed())
}

func createKymaResource(name string) *kymav1beta1.Kyma {
	kyma := kymav1beta1.Kyma{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: skr.KcpNamespace,
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
		if useExistingCluster {
			err = k8sClient.Get(context.TODO(), types.NamespacedName{Namespace: skr.KcpNamespace, Name: kyma.Name}, &v1alpha1.EventingAuth{})
			g.Expect(err).ShouldNot(BeNil())
			g.Expect(errors.IsNotFound(err)).To(BeTrue())
		}
	}, defaultTimeout).Should(Succeed())
}

func createIasCredsSecret(url, username, password string) *corev1.Secret {
	By("Creating IAS credentials secret")
	kubeconfigSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllers.DefaultIasCredsSecretName,
			Namespace: skr.KcpNamespace,
		},
		Data: map[string][]byte{
			"url":      []byte(url),
			"username": []byte(username),
			"password": []byte(password),
		},
	}
	Expect(k8sClient.Create(context.TODO(), &kubeconfigSecret)).Should(Succeed())
	return &kubeconfigSecret
}

func deleteIasCredsSecret() {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllers.DefaultIasCredsSecretName,
			Namespace: skr.KcpNamespace,
		},
	}
	if err := k8sClient.Delete(context.TODO(), s); err != nil && !errors.IsNotFound(err) {
		Fail("Cannot clean up" + s.Namespace + "/" + s.Name + " secret properly")
	}
}
