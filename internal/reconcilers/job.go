package reconcilers

import (
	"context"
	"fmt"

	"github.io/docling-project/docling-operator/api/v1alpha1"
	"github.io/docling-project/docling-operator/internal/templates"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewJobReconciler(client client.Client, scheme *runtime.Scheme) *JobReconciler {
	return &JobReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *JobReconciler) Reconcile(ctx context.Context, doclingServe *v1alpha1.DoclingServe) (bool, error) {
	log := logf.FromContext(ctx)

	if doclingServe.Spec.ArtifactsVolume == nil {
		// Not enabled or not specified. Return with no error and no requeue
		// The Deployment reconciler will add an emptyDir volume.
		log.Info("no artifacts volume specified")
		return false, nil
	}

	// Check that PVC has been bound before even creating job
	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-pvc", doclingModelCache),
		Namespace: doclingServe.Namespace,
	}, &pvc); err != nil {
		return true, err
	}
	if pvc.Status.Phase != corev1.ClaimBound {
		return true, fmt.Errorf("PVC %s is not bound", pvc.Name)
	}

	jobName := fmt.Sprintf("%s-job", doclingModelCache)
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: doclingServe.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, job, func() error {
		type jobExtraValues struct {
			*v1alpha1.DoclingServe
			Image                     string
			PersistentVolumeClaimName string
		}
		extraValues := jobExtraValues{
			DoclingServe:              doclingServe,
			Image:                     "ghcr.io/docling-project/docling-serve-cpu:main",
			PersistentVolumeClaimName: fmt.Sprintf("%s-pvc", doclingModelCache),
		}

		jobtmpl, err := templates.ResourceFromTemplate[jobExtraValues, batchv1.Job](ctx, &extraValues, "job")
		if err != nil {
			return fmt.Errorf("could not get template: %v", err)
		}

		jobtmpl.Spec.Template.Spec.DeepCopyInto(&job.Spec.Template.Spec)

		if err := ctrl.SetControllerReference(doclingServe, job, r.Scheme); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "error reconciling job", "Job.Namespace", job.Namespace, "Job.Name", jobName)
		return true, err
	}
	return false, nil
}
