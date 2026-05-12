/*
Copyright 2026 stubbi.

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

package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	hermesv1 "github.com/stubbi/hermes-operator/api/v1"
)

var _ = Describe("HermesInstance controller", func() {
	const (
		name      = "demo"
		namespace = "default"
		timeout   = 30 * time.Second
		interval  = 250 * time.Millisecond
	)

	AfterEach(func() {
		ctx := context.Background()
		// Delete the HermesInstance first and wait for it to disappear so the
		// controller stops reconciling. Then delete owned resources explicitly
		// (envtest does not run the k8s garbage collector).
		inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
		_ = k8sClient.Delete(ctx, inst)
		Eventually(func() error {
			obj := &hermesv1.HermesInstance{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return fmt.Errorf("HermesInstance %s still exists", name)
		}).Within(timeout).WithPolling(interval).Should(Succeed())

		_ = k8sClient.Delete(ctx, &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
		_ = k8sClient.Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
		_ = k8sClient.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name + "-config", Namespace: namespace}})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: name + "-data", Namespace: namespace}})

		// Wait for the StatefulSet to be gone before the next test starts.
		Eventually(func() error {
			sts := &appsv1.StatefulSet{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, sts)
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return fmt.Errorf("StatefulSet %s still exists", name)
		}).Within(timeout).WithPolling(interval).Should(Succeed())
	})

	It("creates PVC, ConfigMap, Service, and StatefulSet for a new HermesInstance", func() {
		ctx := context.Background()

		inst := &hermesv1.HermesInstance{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: hermesv1.HermesInstanceSpec{
				Image: hermesv1.ImageSpec{
					Repository: "ghcr.io/stubbi/hermes-agent",
					Tag:        "test",
				},
			},
		}
		Expect(k8sClient.Create(ctx, inst)).To(Succeed())

		Eventually(func(g Gomega) {
			pvc := &corev1.PersistentVolumeClaim{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name + "-data", Namespace: namespace}, pvc)).To(Succeed())
			cm := &corev1.ConfigMap{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name + "-config", Namespace: namespace}, cm)).To(Succeed())
			svc := &corev1.Service{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, svc)).To(Succeed())
			sts := &appsv1.StatefulSet{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, sts)).To(Succeed())
			g.Expect(sts.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/stubbi/hermes-agent:test"))
		}).Within(timeout).WithPolling(interval).Should(Succeed())
	})

	It("is idempotent — second reconcile does not change StatefulSet generation", func() {
		ctx := context.Background()

		inst := &hermesv1.HermesInstance{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		Expect(k8sClient.Create(ctx, inst)).To(Succeed())

		var stsGenBefore int64
		Eventually(func(g Gomega) {
			sts := &appsv1.StatefulSet{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, sts)).To(Succeed())
			stsGenBefore = sts.Generation
		}).Within(timeout).WithPolling(interval).Should(Succeed())

		Eventually(func() error {
			var cur hermesv1.HermesInstance
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &cur); err != nil {
				return err
			}
			if cur.Annotations == nil {
				cur.Annotations = map[string]string{}
			}
			cur.Annotations["test.example.com/poke"] = time.Now().String()
			return k8sClient.Update(ctx, &cur)
		}).Within(timeout).WithPolling(interval).Should(Succeed())

		time.Sleep(2 * time.Second)

		sts := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, sts)).To(Succeed())
		Expect(sts.Generation).To(Equal(stsGenBefore), "STS generation must not bump on no-op reconcile")
	})
})
