package reconcilers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.io/opdev/docling-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewServiceReconciler(client client.Client, scheme *runtime.Scheme) *ServiceReconciler {
	return &ServiceReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	log := logf.FromContext(ctx)
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: doclingServ.Name + "-service", Namespace: doclingServ.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		labels := labelsForDocling(doclingServ.Name)
		service.Labels = labels
		service.Spec.Selector = labels
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Port:       5001,
				TargetPort: intstr.FromInt32(5001),
			},
		}
		_ = ctrl.SetControllerReference(doclingServ, service, r.Scheme)
		return nil
	})
	if err != nil {
		log.Error(err, "Error reconciling Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		return true, err
	}

	log.Info("Successfully reconciled Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
	return false, nil
}
