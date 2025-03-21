package reconcilers

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	"github.io/opdev/docling-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RouteReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewRouteReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *RouteReconciler {
	return &RouteReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *RouteReconciler) Reconcile(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doclingServ.Name + "-route",
			Namespace: doclingServ.Namespace,
		},
	}

	err := r.Client.Get(ctx, types.NamespacedName{Name: route.GetName(), Namespace: route.GetNamespace()}, route)

	// if route exists, update it
	if err == nil {
		if err := r.Client.Update(ctx, route); err != nil {
			r.Log.Error(err, "Failed to update Route", "Route", route.GetName())
			return false, err
		}
		// if route doesn't exist, create it
	} else if errors.IsNotFound(err) {
		route = r.routeForDoclingServ(doclingServ)
		if err := r.Client.Create(ctx, route); err != nil {
			r.Log.Error(err, "Failed to create Route", "Route", route.GetName())
			return false, err
		}
		_ = ctrl.SetControllerReference(doclingServ, route, r.Scheme)
	} else {
		return false, err
	}

	return false, nil
}

func (r *RouteReconciler) routeForDoclingServ(doclingServ *v1alpha1.DoclingServ) *routev1.Route {
	labels := labelsForDocling(doclingServ.Name)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doclingServ.Name + "-route",
			Namespace: doclingServ.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
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
		},
	}
}
