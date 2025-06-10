package reconcilers

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.io/docling-project/docling-operator/api/v1alpha1"
)

const (
	defaultVolumeSize = 16
)

type VolumeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewVolumeReconciler(client client.Client, scheme *runtime.Scheme) *VolumeReconciler {
	return &VolumeReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *VolumeReconciler) Reconcile(ctx context.Context, doclingServe *v1alpha1.DoclingServe) (bool, error) {
	log := logf.FromContext(ctx)

	artifactsVolume := doclingServe.Spec.ArtifactsVolume
	if artifactsVolume == nil {
		// Not enabled or not specified. Return with no error and no requeue
		// The Deployment reconciler will add an emptyDir volume.
		log.Info("no artifacts volume specified")
		return false, nil
	}

	pvcName := fmt.Sprintf("%s-pvc", doclingModelCache)
	pvc := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: pvcName, Namespace: doclingServe.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		if pvc.Spec.VolumeName != "" {
			// We currently don't support using an explicit volume,
			// so this means that one was bound to the PVC. Just
			// return with no error.
			return nil
		}

		pvcResources := v1.VolumeResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceStorage: doclingServe.Spec.ArtifactsVolume.PVCSize,
			},
		}
		pvc.Spec = v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
				v1.ReadOnlyMany,
			},
			Resources: pvcResources,
		}

		if doclingServe.Spec.ArtifactsVolume.StorageClassName != "" {
			pvc.Spec.StorageClassName = &doclingServe.Spec.ArtifactsVolume.StorageClassName
		}

		if err := ctrl.SetControllerReference(doclingServe, pvc, r.Scheme); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "Error reconciling Volume", "PersistentVolumeClaim.Namespace", pvc.Namespace, "PersistentVolumeClaim.Name", pvcName)
		return true, err
	}
	return false, nil
}
