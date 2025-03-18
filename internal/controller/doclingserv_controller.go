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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.io/opdev/docling-operator/api/v1alpha1"
	"github.io/opdev/docling-operator/internal/reconcilers"
)

var log = logf.Log.WithName("controller_doclingserv")

// DoclingServReconciler reconciles a DoclingServ object
type DoclingServReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=docling.github.io,resources=doclingservs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=docling.github.io,resources=doclingservs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=docling.github.io,resources=doclingservs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods;services,verbs=get;list;watch;
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DoclingServ object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *DoclingServReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logf.FromContext(ctx, "Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling DoclingServ")

	currentDoclingServ := &v1alpha1.DoclingServ{}
	err := r.Get(ctx, req.NamespacedName, currentDoclingServ)
	if err != nil {
		if errors.IsNotFound(err) {
			// Error reading the object - requeue the request.
			return reconcile.Result{}, nil
		}
	}

	resourceReconcilers := []reconcilers.Reconciler{
		reconcilers.NewDeploymentReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewServiceReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewRouteReconciler(r.Client, reqLogger, r.Scheme),
	}

	requeueResult := false
	var errResult error = nil
	doclingServ := currentDoclingServ.DeepCopy()
	for _, r := range resourceReconcilers {
		reque, err := r.Reconcile(ctx, doclingServ)
		if err != nil {
			// Only capture the first error
			log.Error(err, "requeuing with error")
			errResult = err
		}
		requeueResult = requeueResult || reque
	}

	return ctrl.Result{Requeue: requeueResult}, errResult
}

// SetupWithManager sets up the controller with the Manager.
func (r *DoclingServReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DoclingServ{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&routev1.Route{}).
		Complete(r)
}
