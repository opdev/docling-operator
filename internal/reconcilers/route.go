package reconcilers

import (
	"context"

	routev1 "github.com/openshift/api/route/v1"
	"github.io/opdev/docling-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type RouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewRouteReconciler(client client.Client, scheme *runtime.Scheme) *RouteReconciler {
	return &RouteReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *RouteReconciler) Reconcile(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	if doclingServ.Spec.Route != nil && doclingServ.Spec.Route.Enabled {
		return r.createOrUpdate(ctx, doclingServ)
	}

	return r.delete(ctx, doclingServ)
}

func (r *RouteReconciler) createOrUpdate(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	log := logf.FromContext(ctx)
	route := &routev1.Route{ObjectMeta: metav1.ObjectMeta{Name: doclingServ.Name + "-route", Namespace: doclingServ.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, route, func() error {
		labels := labelsForDocling(doclingServ.Name)
		route.Labels = labels
		route.Spec = routev1.RouteSpec{
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: doclingServ.Name + "-service",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		}
		_ = ctrl.SetControllerReference(doclingServ, route, r.Scheme)
		return nil
	})
	if err != nil {
		log.Error(err, "Error creating/updating Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		return true, err
	}

	log.Info("Successfully created/updated Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
	return false, nil
}

func (r *RouteReconciler) delete(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	log := logf.FromContext(ctx)
	route := &routev1.Route{ObjectMeta: metav1.ObjectMeta{Name: doclingServ.Name + "-route", Namespace: doclingServ.Namespace}}
	if err := r.Get(ctx, types.NamespacedName{Name: doclingServ.Name + "-route", Namespace: doclingServ.Namespace}, route); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Error deleting Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		return true, err
	} else if errors.IsNotFound(err) {
		return false, nil
	}

	if err := r.Delete(ctx, route); err != nil {
		log.Error(err, "Error deleting Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		return true, err
	}

	log.Info("Successfully deleted Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
	return false, nil
}
