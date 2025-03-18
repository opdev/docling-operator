package reconcilers

import (
	"context"

	"github.com/go-logr/logr"
	"github.io/opdev/docling-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewDeploymentReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *DeploymentReconciler {
	return &DeploymentReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, doclingServ *v1alpha1.DoclingServ) (bool, error) {
	foundDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: doclingServ.Name, Namespace: doclingServ.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Deployment", "Deployment", doclingServ.Name)
		deployment := r.deploymentForDoclingServ(doclingServ)

		err = r.Create(ctx, deployment)
		if err != nil {
			r.Log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return true, err
		}
	}

	return false, nil
}

func (r *DeploymentReconciler) deploymentForDoclingServ(doclingServ *v1alpha1.DoclingServ) *appsv1.Deployment {
	labels := labelsForDocling(doclingServ.Name)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doclingServ.Name,
			Namespace: doclingServ.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &doclingServ.Spec.ReplicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: doclingServ.Spec.ImageReference,
						Name:  "docling-serv",
						Command: []string{
							"docling-serve",
							"run",
						},
						ImagePullPolicy: corev1.PullIfNotPresent,
					},
					},
				},
			},
		},
	}

	if doclingServ.Spec.EnableUI {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{{
			Name:  "DOCLING_SERVE_ENABLE_UI",
			Value: "true",
		}}...)
	}

	_ = ctrl.SetControllerReference(doclingServ, deployment, r.Scheme)

	return deployment
}

func labelsForDocling(name string) map[string]string {
	return map[string]string{"app": "docling-serve", "doclingserv_cr": name}
}
