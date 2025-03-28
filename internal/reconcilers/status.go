package reconcilers

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	"github.io/opdev/docling-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deploymentName = "docling-serv"
)

type StatusReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewStatusReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *StatusReconciler {
	return &StatusReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *StatusReconciler) Reconcile(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {

	log := r.Log.WithValues("Status.ObservedGeneration", doclingServ.Generation)

	origDoclingServ := doclingServ.DeepCopy()
	doclingServ.Status.DoclingObservedGeneration = doclingServ.Generation

	var requeue bool
	var err error

	// Always try to commit the status
	defer r.commitStatus(ctx, doclingServ, log)

	// Update observed generation
	doclingServ.Status.DoclingObservedGeneration = doclingServ.Generation

	// Image reference (from deployment)
	requeue, err = r.reconcileDoclingServImageReference(ctx, doclingServ, log)
	if requeue || err != nil {
		log.Error(err, "DoclingServImageReference")
		return requeue, err
	}

	// UI enabled status (from deployment)
	requeue, err = r.reconcileDoclingUIEnabledStatus(ctx, doclingServ, log)
	if requeue || err != nil {
		log.Error(err, "DoclingUIEnabledStatus")
		return requeue, err
	}

	// Deployment status
	requeue, err = r.reconcileDoclingDeploymentStatus(ctx, doclingServ, log)
	if requeue || err != nil {
		log.Error(err, "DoclingDeploymentStatus")
		return requeue, err
	}

	// Service status
	requeue, err = r.reconcileDoclingServiceStatus(ctx, doclingServ, log)
	if requeue || err != nil {
		log.Error(err, "DoclingServiceStatus")
		return requeue, err
	}

	// Route status
	requeue, err = r.reconcileDoclingRouteStatus(ctx, doclingServ, log)
	if requeue || err != nil {
		log.Error(err, "DoclingRouteStatus")
		return requeue, err
	}

	// Number of ready replicas
	requeue, err = r.reconcileDoclingReadyReplicas(ctx, doclingServ, log)
	if requeue || err != nil {
		log.Error(err, "DoclingReadyReplicas")
		return requeue, err
	}

	if equality.Semantic.DeepEqual(doclingServ.Status, origDoclingServ.Status) {
		log.Info("Successfully reconciled Status", "Status", doclingServ.Status)
		return false, nil
	}

	return true, nil
}

func (r *StatusReconciler) commitStatus(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) {
	err := r.Client.Status().Update(ctx, doclingServ)
	if err != nil && apierrors.IsConflict(err) {
		log.Info("conflict updating doclingServ status")
		return
	}
	if err != nil {
		log.Error(err, "failed to update doclingServ status")
		return
	}
	log.Info("updated doclingServ status")
}

func (r *StatusReconciler) reconcileDoclingServImageReference(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) (bool, error) {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: doclingServ.Namespace}, &deployment)
	if err != nil {
		log.Error(err, "failed to get doclingServ deployment")
		return true, err
	}
	doclingServ.Status.DoclingServImage = deployment.Spec.Template.Spec.Containers[0].Image
	return false, nil
}

func (r *StatusReconciler) reconcileDoclingUIEnabledStatus(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) (bool, error) {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: doclingServ.Namespace}, &deployment)
	if err != nil {
		log.Error(err, "failed to get doclingServ deployment")
		return true, err
	}

	doclingServ.Status.DoclingServUIEnabled = string(metav1.ConditionFalse)
	cmd := deployment.Spec.Template.Spec.Containers[0].Command
	for _, flag := range cmd {
		if strings.Contains(flag, "--enable-ui") {
			doclingServ.Status.DoclingServUIEnabled = string(metav1.ConditionTrue)
		}
	}

	env := deployment.Spec.Template.Spec.Containers[0].Env
	for _, env := range env {
		if env.Name == "DOCLING_SERVE_ENABLE_UI" && env.Value == "true" {
			doclingServ.Status.DoclingServUIEnabled = string(metav1.ConditionTrue)
		}
	}

	return false, nil
}

func (r *StatusReconciler) reconcileDoclingDeploymentStatus(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) (bool, error) {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: doclingServ.Namespace}, &deployment)
	if err != nil {
		log.Error(err, "failed to get doclingServ deployment")
		return true, err
	}

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
			meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
		}
	}

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
		meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
	}
	return false, nil
}

func (r *StatusReconciler) reconcileDoclingServiceStatus(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) (bool, error) {
	service := corev1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: doclingServ.Name + "-service", Namespace: doclingServ.Namespace}, &service)
	if err != nil {
		log.Error(err, "failed to get doclingServ service")
		return true, err
	}

	if service.Status.String() != "" {
		condition := metav1.Condition{
			Type:               "ServiceCreated",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: service.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "ServiceCreated",
			Message:            "The docling service was created successfully",
		}
		meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
	}

	if len(service.Status.LoadBalancer.Ingress) > 0 {
		condition := metav1.Condition{
			Type:               "LoadBalancerAssigned",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: service.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "LoadBalancerAssigned",
			Message:            "A LoadBalancer has been assigned to the docling service",
		}
		meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
	}

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
		meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
	}

	return false, nil
}

func (r *StatusReconciler) reconcileDoclingRouteStatus(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) (bool, error) {
	route := routev1.Route{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: doclingServ.Name + "-route", Namespace: doclingServ.Namespace}, &route)
	if err != nil {
		log.Error(err, "failed to get doclingServ route")
		return true, err
	}

	if route.Status.String() != "" {
		condition := metav1.Condition{
			Type:               "RouteCreated",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: route.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             "RouteCreated",
			Message:            "A docling route was created successfully",
		}
		meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
	}

	if len(route.Status.Ingress) > 0 && len(route.Status.Ingress[0].Conditions) > 0 {
		routeCondition := route.Status.Ingress[0].Conditions[len(route.Status.Ingress[0].Conditions)-1]
		condition := metav1.Condition{
			Type:               string(routeCondition.Type),
			Status:             metav1.ConditionStatus(routeCondition.Status),
			ObservedGeneration: route.Generation,
			LastTransitionTime: metav1.Time{},
			Reason:             routeCondition.Reason,
			Message:            routeCondition.Message,
		}
		meta.SetStatusCondition(&doclingServ.Status.Conditions, condition)
	}

	return false, nil
}

func (r *StatusReconciler) reconcileDoclingReadyReplicas(ctx context.Context, doclingServ *v1alpha1.DoclingServ, log logr.Logger) (bool, error) {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: doclingServ.Namespace}, &deployment)
	if err != nil {
		log.Error(err, "failed to get doclingServ deployment")
		return true, err
	}
	doclingServ.Status.DoclingReadyReplicas = deployment.Status.ReadyReplicas

	return false, nil
}
