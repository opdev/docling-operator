package reconcilers

import (
	"context"
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	"github.io/docling-project/docling-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type StatusReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewStatusReconciler(client client.Client, scheme *runtime.Scheme) *StatusReconciler {
	return &StatusReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *StatusReconciler) Reconcile(ctx context.Context, doclingServe *v1alpha1.DoclingServe) (bool, error) {
	log := logf.FromContext(ctx, "Status.ObservedGeneration", doclingServe.Generation)
	ctx = logf.IntoContext(ctx, log)
	doclingServe.Status.ObservedGeneration = doclingServe.Generation

	var err error
	var requeue bool

	// Always try to commit the status and propagate the error from CommitStatus.
	defer func() {
		err = r.commitStatus(ctx, doclingServe)
		if err != nil {
			requeue = true
		}
	}()

	// Update deployment status
	r.reconcileDoclingDeploymentStatus(ctx, doclingServe)

	// Update service status
	r.reconcileDoclingServiceStatus(ctx, doclingServe)

	// Update route status
	r.reconcileDoclingRouteStatus(ctx, doclingServe)

	return requeue, err
}

func (r *StatusReconciler) commitStatus(ctx context.Context, doclingServe *v1alpha1.DoclingServe) error {
	log := logf.FromContext(ctx)
	err := r.Client.Status().Update(ctx, doclingServe)
	if err != nil && apierrors.IsConflict(err) {
		log.Info("conflict updating doclingServe status")
		return err
	}
	if err != nil {
		log.Error(err, "failed to update doclingServe status")
		return err
	}
	log.Info("updated doclingServe status")
	return err
}

func (r *StatusReconciler) reconcileDoclingDeploymentStatus(ctx context.Context, doclingServe *v1alpha1.DoclingServe) {
	log := logf.FromContext(ctx)
	deployment := appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-deployment", doclingServe.Name), Namespace: doclingServe.Namespace}, &deployment); err != nil {
		log.Error(err, "failed to get doclingServe deployment")
		condition := metav1.Condition{
			Type:               "DeploymentCreated",
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: deployment.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "DeploymentStatusError",
			Message:            err.Error(),
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
		return
	}

	// Set created status
	if deployment.Status.String() != "" {
		condition := metav1.Condition{
			Type:               "DeploymentCreated",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: deployment.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "DeploymentCreated",
			Message:            "The docling deployment was created successfully",
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}

	// Set availability status
	for _, deployCondition := range deployment.Status.Conditions {
		if deployCondition.Type == appsv1.DeploymentAvailable {
			condition := metav1.Condition{
				Type:               "DeploymentAvailable",
				Status:             metav1.ConditionStatus(deployCondition.Status),
				ObservedGeneration: deployment.Generation,
				LastTransitionTime: metav1.Time{},
				Reason:             deployCondition.Reason,
				Message:            deployCondition.Message,
			}
			meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
		}
	}

	// Set misc statuses for troubleshooting
	if len(deployment.Status.Conditions) > 0 {
		deployCondition := deployment.Status.Conditions[len(deployment.Status.Conditions)-1]
		condition := metav1.Condition{
			Type:               string(deployCondition.Type),
			Status:             metav1.ConditionStatus(deployCondition.Status),
			ObservedGeneration: deployment.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             deployCondition.Reason,
			Message:            deployCondition.Message,
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}
}

func (r *StatusReconciler) reconcileDoclingServiceStatus(ctx context.Context, doclingServe *v1alpha1.DoclingServe) {
	log := logf.FromContext(ctx)
	service := corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-service", doclingServe.Name), Namespace: doclingServe.Namespace}, &service); err != nil {
		log.Error(err, "failed to get doclingServe service")
		condition := metav1.Condition{
			Type:               "ServiceCreated",
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: service.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "ServiceStatusError",
			Message:            err.Error(),
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
		return
	}

	// Set created status
	if service.Status.String() != "" {
		condition := metav1.Condition{
			Type:               "ServiceCreated",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: service.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "ServiceCreated",
			Message:            "The docling service was created successfully",
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}

	// Set loadbalancer status
	if len(service.Status.LoadBalancer.Ingress) > 0 {
		condition := metav1.Condition{
			Type:               "LoadBalancerAssigned",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: service.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "LoadBalancerAssigned",
			Message:            "A LoadBalancer has been assigned to the docling service",
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}

	// Set misc statuses for troubleshooting
	if len(service.Status.Conditions) > 0 {
		serviceCondition := service.Status.Conditions[len(service.Status.Conditions)-1]
		condition := metav1.Condition{
			Type:               serviceCondition.Type,
			Status:             serviceCondition.Status,
			ObservedGeneration: service.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             serviceCondition.Reason,
			Message:            serviceCondition.Message,
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}
}

func (r *StatusReconciler) reconcileDoclingRouteStatus(ctx context.Context, doclingServe *v1alpha1.DoclingServe) {
	log := logf.FromContext(ctx)
	if doclingServe.Spec.Route != nil && !doclingServe.Spec.Route.Enabled {
		// Route is not enabled, so write a condition as such and return
		condition := metav1.Condition{
			Type:               "RouteCreated",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{},
			Reason:             "RouteDisabled",
			Message:            "A docling route is disabled",
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
		return
	}

	route := routev1.Route{}
	if err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-route", doclingServe.Name), Namespace: doclingServe.Namespace}, &route); err != nil {
		log.Error(err, "failed to get doclingServe route")
		condition := metav1.Condition{
			Type:               "RouteCreated",
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: route.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "RouteStatusError",
			Message:            err.Error(),
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
		return
	}

	// Set created status
	if route.Status.String() != "" {
		condition := metav1.Condition{
			Type:               "RouteCreated",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: route.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "RouteCreated",
			Message:            "A docling route was created successfully",
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}

	// Set ingress status
	if len(route.Status.Ingress) > 0 && len(route.Status.Ingress[0].Conditions) > 0 {
		routeCondition := route.Status.Ingress[0].Conditions[len(route.Status.Ingress[0].Conditions)-1]
		// Avoid temporary empty string that results in an error when updating status
		if routeCondition.Reason == "" {
			routeCondition.Reason = "Unknown"
		}
		condition := metav1.Condition{
			Type:               string(routeCondition.Type),
			Status:             metav1.ConditionStatus(routeCondition.Status),
			ObservedGeneration: route.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             routeCondition.Reason,
			Message:            routeCondition.Message,
		}
		meta.SetStatusCondition(&doclingServe.Status.Conditions, condition)
	}
}
