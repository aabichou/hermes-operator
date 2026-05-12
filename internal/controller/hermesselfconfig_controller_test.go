package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hermesv1 "github.com/stubbi/hermes-operator/api/v1"
)

var _ = Describe("HermesSelfConfig controller", func() {
	const (
		ns      = "default"
		timeout = 30 * time.Second
		poll    = 200 * time.Millisecond
	)

	AfterEach(func() {
		ctx := context.Background()
		for _, name := range []string{"deny-target", "happy-target"} {
			_ = k8sClient.Delete(ctx, &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}})
		}
		scs := &hermesv1.HermesSelfConfigList{}
		_ = k8sClient.List(ctx, scs, &client.ListOptions{Namespace: ns})
		for i := range scs.Items {
			_ = k8sClient.Delete(ctx, &scs.Items[i])
		}
	})

	It("denies a SelfConfig whose parent has selfConfigure.enabled=false", func() {
		ctx := context.Background()
		parent := &hermesv1.HermesInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "deny-target", Namespace: ns},
			Spec: hermesv1.HermesInstanceSpec{
				Image: hermesv1.ImageSpec{
					Repository: "ghcr.io/stubbi/hermes-agent",
					Tag:        "test",
				},
				// SelfConfigure.Enabled left nil/false on purpose.
			},
		}
		Expect(k8sClient.Create(ctx, parent)).To(Succeed())

		sc := &hermesv1.HermesSelfConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "deny-this", Namespace: ns},
			Spec: hermesv1.HermesSelfConfigSpec{
				InstanceRef: "deny-target",
				AddSkills:   []hermesv1.SelfConfigSkill{{Source: "git+x"}},
			},
		}
		Expect(k8sClient.Create(ctx, sc)).To(Succeed())

		Eventually(func(g Gomega) {
			got := &hermesv1.HermesSelfConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "deny-this", Namespace: ns}, got)).To(Succeed())
			g.Expect(got.Status.Phase).To(Equal(hermesv1.SelfConfigPhaseDenied))
			g.Expect(got.Status.DenyReason).To(ContainSubstring("selfconfig disabled"))
		}).Within(timeout).WithPolling(poll).Should(Succeed())
	})
})
