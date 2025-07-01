package reconcilers

import (
	"context"
	"fmt"
	"strconv"

	"github.io/docling-project/docling-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
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
				ServiceAccountName: serviceAccountName,
				Containers: []corev1.Container{
					{
						Image: doclingServe.Spec.APIServer.Image,
						Name:  "docling-serve",
						Command: []string{
							"docling-serve",
							"run",
						},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 5001,
								Name:          "http",
								Protocol:      corev1.ProtocolTCP,
							},
						},
						ImagePullPolicy: corev1.PullIfNotPresent,
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/health",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: 3,
							TimeoutSeconds:      4,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							FailureThreshold:    5,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/health",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      2,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    3,
						},
					},
				},
			},
		}

		// Collect env vars and sources and do one append at the end
		additionalEnvVars := make([]corev1.EnvVar, 0, 10)
		additionalEnvFrom := make([]corev1.EnvFromSource, 0, 10)

		if doclingServe.Spec.APIServer.EnableUI {
			additionalEnvVars = append(additionalEnvVars, corev1.EnvVar{
				Name:  "DOCLING_SERVE_ENABLE_UI",
				Value: "true",
			})
		}

		if doclingServe.Spec.Engine.Local != nil {
			additionalEnvVars = append(additionalEnvVars, corev1.EnvVar{
				Name:  "DOCLING_SERVE_ENG_LOC_NUM_WORKERS",
				Value: strconv.Itoa(int(doclingServe.Spec.Engine.Local.NumWorkers)),
			})
		}

		if doclingServe.Spec.Engine.KFP != nil {
			additionalEnvVars = append(additionalEnvVars,
				[]corev1.EnvVar{
					{
						Name:  "DOCLING_SERVE_ENG_KFP_ENDPOINT",
						Value: doclingServe.Spec.Engine.KFP.Endpoint,
					},
					{
						Name:  "DOCLING_SERVE_ENG_KIND",
						Value: "kfp",
					},
				}...)
		}

		if len(doclingServe.Spec.APIServer.ConfigMapName) > 0 {
			additionalEnvFrom = append(additionalEnvFrom, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: doclingServe.Spec.APIServer.ConfigMapName},
					Optional:             new(bool),
				},
			})
		}

		if doclingServe.Spec.APIServer.Resources != nil {
			deployment.Spec.Template.Spec.Containers[0].Resources = *doclingServe.Spec.APIServer.Resources
		}

		if doclingServe.Spec.ArtifactsVolume == nil {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: doclingModelCache,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium:    "",
						SizeLimit: resource.NewScaledQuantity(defaultVolumeSize, resource.Giga),
					},
				},
			})
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      doclingModelCache,
				ReadOnly:  true,
				MountPath: defaultArtifactsPath,
			})
		}

		if doclingServe.Spec.ArtifactsVolume != nil {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: doclingModelCache,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-pvc", doclingModelCache),
						ReadOnly:  true,
					},
				},
			})

			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      doclingModelCache,
				ReadOnly:  true,
				MountPath: defaultArtifactsPath,
			})
		}
		additionalEnvVars = append(additionalEnvVars,
			corev1.EnvVar{
				Name:  "DOCLING_SERVER_ARTIFACTS_PATH",
				Value: defaultArtifactsPath,
			})

		if len(additionalEnvVars) > 0 {
			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, additionalEnvVars...)
		}
		if len(additionalEnvFrom) > 0 {
			deployment.Spec.Template.Spec.Containers[0].EnvFrom = append(deployment.Spec.Template.Spec.Containers[0].EnvFrom, additionalEnvFrom...)
		}
		_ = ctrl.SetControllerReference(doclingServe, deployment, r.Scheme)

		return nil
	}); err != nil {
		log.Error(err, "Error reconciling Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return true, err
	}

	log.Info("Successfully reconciled Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

	return false, nil
}
