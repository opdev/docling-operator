package reconcilers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	log.Info("external artifacts volume not yet supported")
	return false, fmt.Errorf("not yet implemented")
}
