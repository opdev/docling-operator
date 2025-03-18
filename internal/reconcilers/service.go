package reconcilers

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	"github.io/opdev/docling-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewServiceReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *ServiceReconciler {
	return &ServiceReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doclingServ.Name + "-service",
			Namespace: doclingServ.Namespace,
		},
	}

	err := r.Client.Get(ctx, types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, service)

	if err == nil {
		if err := r.Client.Update(ctx, service); err != nil {
			r.Log.Error(err, "Failed to update Service", "Service", service.GetName())
			return false, err
		}
	} else if errors.IsNotFound(err) {
		service = r.serviceForDoclingServ(doclingServ)
		if err := r.Client.Create(ctx, service); err != nil {
			r.Log.Error(err, "Failed to create Service", "Service", service.GetName())
			return false, err
		}
		_ = ctrl.SetControllerReference(doclingServ, service, r.Scheme)
	} else {
		return false, err
	}

	return false, nil
}

func (r *ServiceReconciler) serviceForDoclingServ(doclingServ *v1alpha1.DoclingServ) *corev1.Service {
	labels := labelsForDocling(doclingServ.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doclingServ.Name + "-service",
			Namespace: doclingServ.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       5001,
					TargetPort: intstr.FromInt32(5001),
				},
			},
		},
	}
}
