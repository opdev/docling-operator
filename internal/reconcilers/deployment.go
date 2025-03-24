package reconcilers

import (
	"context"

	"github.com/go-logr/logr"
	"github.io/opdev/docling-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "docling-serv", Namespace: doclingServ.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := labelsForDocling(doclingServ.Name)
		if deployment.ObjectMeta.CreationTimestamp.IsZero() {
			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
		}

		deployment.Spec.Replicas = &doclingServ.Spec.ReplicaCount
		deployment.Spec.Template = corev1.PodTemplateSpec{
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
		}

		if doclingServ.Spec.EnableUI {
			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{{
				Name:  "DOCLING_SERVE_ENABLE_UI",
				Value: "true",
			}}...)
		}

		_ = ctrl.SetControllerReference(doclingServ, deployment, r.Scheme)

		return nil
	})
	if err != nil {
		r.Log.Error(err, "Error reconciling Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return true, err
	}

	r.Log.Info("Successfully reconciled Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

	return false, nil
}

func labelsForDocling(name string) map[string]string {
	return map[string]string{"app": "docling-serve", "doclingserv_cr": name}
}
