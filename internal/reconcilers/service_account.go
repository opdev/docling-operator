package reconcilers

import (
	"context"

	"github.io/opdev/docling-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type ServiceAccountReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const serviceAccountName = "docling-serve"

func NewServiceAccountReconciler(client client.Client, scheme *runtime.Scheme) *ServiceAccountReconciler {
	return &ServiceAccountReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *ServiceAccountReconciler) Reconcile(ctx context.Context, doclingServe *v1alpha1.DoclingServe) (bool, error) {
	log := logf.FromContext(ctx)
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: doclingServe.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		serviceAccount.Labels = labelsForDocling(doclingServe.Name)
		_ = ctrl.SetControllerReference(doclingServe, serviceAccount, r.Scheme)
		return nil
	})
	if err != nil {
		log.Error(err, "Error creating ServiceAccount", "ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
		return true, err
	}

	log.Info("Successfully created ServiceAccount", "ServiceAccount.Namespace", serviceAccount.Namespace, "ServiceAccount.Name", serviceAccount.Name)
	return false, nil
}
