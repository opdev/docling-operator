package reconcilers

import (
	"context"
	"strconv"

	"github.io/opdev/docling-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewDeploymentReconciler(client client.Client, scheme *runtime.Scheme) *DeploymentReconciler {
	return &DeploymentReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, doclingServe *v1alpha1.DoclingServe) (bool, error) {
	log := logf.FromContext(ctx)

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: doclingServe.Name + "-deployment", Namespace: doclingServe.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := labelsForDocling(doclingServe.Name)
		if deployment.CreationTimestamp.IsZero() {
			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
		}

		deployment.Spec.Replicas = &doclingServe.Spec.APIServer.Instances
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: doclingServe.Spec.APIServer.Image,
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

		if doclingServe.Spec.APIServer.EnableUI {
			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{{
				Name:  "DOCLING_SERVE_ENABLE_UI",
				Value: "true",
			}}...)
		}

		if doclingServe.Spec.Engine.Local != nil {
			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{{
				Name:  "DOCLING_SERVE_ENG_LOC_NUM_WORKERS",
				Value: strconv.Itoa(int(doclingServe.Spec.Engine.Local.NumWorkers)),
			}}...)
		}

		if doclingServe.Spec.Engine.KFP != nil {
			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, []corev1.EnvVar{{
				Name:  "DOCLING_SERVE_ENG_KFP_ENDPOINT",
				Value: doclingServe.Spec.Engine.KFP.Endpoint,
			}}...)
		}

		if len(doclingServe.Spec.APIServer.ConfigMapName) > 0 {
			deployment.Spec.Template.Spec.Containers[0].EnvFrom = append(deployment.Spec.Template.Spec.Containers[0].EnvFrom, []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: doclingServe.Spec.APIServer.ConfigMapName},
					Optional:             new(bool),
				},
			}}...)
		}

		_ = ctrl.SetControllerReference(doclingServe, deployment, r.Scheme)

		return nil
	})
	if err != nil {
		log.Error(err, "Error reconciling Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return true, err
	}

	log.Info("Successfully reconciled Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

	return false, nil
}

func labelsForDocling(name string) map[string]string {
	return map[string]string{"app": "docling-serve", "doclingserve_cr": name}
}
