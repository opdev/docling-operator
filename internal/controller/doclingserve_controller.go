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

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.io/docling-project/docling-operator/api/v1alpha1"
	"github.io/docling-project/docling-operator/internal/reconcilers"
)

var log = logf.Log.WithName("controller_doclingserve")

// DoclingServeReconciler reconciles a DoclingServe object
type DoclingServeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=docling.github.io,resources=doclingserves,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=docling.github.io,resources=doclingserves/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=docling.github.io,resources=doclingserves/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods;services;serviceaccounts;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DoclingServe object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *DoclingServeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logf.FromContext(ctx, "Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling DoclingServe")

	ctx = logf.IntoContext(ctx, reqLogger)

	currentDoclingServe := &v1alpha1.DoclingServe{}
	err := r.Get(ctx, req.NamespacedName, currentDoclingServe)
	if err != nil {
		if errors.IsNotFound(err) {
			// Error reading the object - requeue the request.
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	resourceReconcilers := []reconcilers.Reconciler{
		reconcilers.NewServiceAccountReconciler(r.Client, r.Scheme),
		reconcilers.NewVolumeReconciler(r.Client, r.Scheme),
		reconcilers.NewJobReconciler(r.Client, r.Scheme),
		reconcilers.NewDeploymentReconciler(r.Client, r.Scheme),
		reconcilers.NewServiceReconciler(r.Client, r.Scheme),
		reconcilers.NewRouteReconciler(r.Client, r.Scheme),
	}
	statusReconciler := reconcilers.NewStatusReconciler(r.Client, r.Scheme)

	doclingServe := currentDoclingServe.DeepCopy()
	var reconcileErr error
	requeue := false

	for _, r := range resourceReconcilers {
		shouldRequeue, err := r.Reconcile(ctx, doclingServe)
		if err != nil {
			reconcileErr = err
			break
		}
		if shouldRequeue {
			requeue = true
		}
	}

	if _, err := statusReconciler.Reconcile(ctx, doclingServe); err != nil {
		if reconcileErr == nil {
			reconcileErr = err
		} else {
			log.Error(err, "additionally, failed to update status")
		}
	}

	return ctrl.Result{Requeue: requeue}, reconcileErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *DoclingServeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DoclingServe{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Owns(&routev1.Route{}).
		Complete(r)
}
