package reconcilers

import (
	"context"

	"github.io/docling-project/docling-operator/api/v1alpha1"
)

type Reconciler interface {
	Reconcile(ctx context.Context, doclingServe *v1alpha1.DoclingServe) (bool, error)
}
