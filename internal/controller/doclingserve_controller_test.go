/*
Copyright 2025.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	doclinggithubiov1alpha1 "github.io/docling-project/docling-operator/api/v1alpha1"
)

var _ = Describe("DoclingServe Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		doclingserve := &doclinggithubiov1alpha1.DoclingServe{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind DoclingServe")
			err := k8sClient.Get(ctx, typeNamespacedName, doclingserve)
			if err != nil && errors.IsNotFound(err) {
				resource := &doclinggithubiov1alpha1.DoclingServe{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: doclinggithubiov1alpha1.DoclingServeSpec{
						APIServer: &doclinggithubiov1alpha1.APIServer{
							Image: "registry/image:tag",
						},
						Engine: &doclinggithubiov1alpha1.Engine{
							Local: &doclinggithubiov1alpha1.Local{},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &doclinggithubiov1alpha1.DoclingServe{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance DoclingServe")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &DoclingServeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
		It("should fail CRD validation", func() {
			By("creating a custom resource for the Kind DoclingServe that includes both `Local` and `KFP` resources")
			err := k8sClient.Get(ctx, typeNamespacedName, doclingserve)
			if err != nil && errors.IsNotFound(err) {
				resource := &doclinggithubiov1alpha1.DoclingServe{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: doclinggithubiov1alpha1.DoclingServeSpec{
						APIServer: &doclinggithubiov1alpha1.APIServer{
							Image: "registry/image:tag",
						},
						Engine: &doclinggithubiov1alpha1.Engine{
							Local: &doclinggithubiov1alpha1.Local{},
							KFP:   &doclinggithubiov1alpha1.KFP{},
						},
					},
				}
				err := k8sClient.Create(ctx, resource)
				Expect(err).To(HaveOccurred())
			}
		})
	})
})
