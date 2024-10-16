package common

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/zncdatadev/operator-go/pkg/builder"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InternalSharedSecretEnvName = "INTERNAL_SHARED_SECRET"
)

func getInternalSharedSecretName(clusterName string) string {
	return clusterName + "-internal-shared-secret"
}

var _ builder.ConfigBuilder = &InternalSharedSecretBuilder{}

type InternalSharedSecretBuilder struct {
	builder.SecretBuilder
}

func (b *InternalSharedSecretBuilder) Build(ctx context.Context) (ctrlclient.Object, error) {
	randomData := make([]byte, 512)
	_, err := rand.Read(randomData)
	if err != nil {
		return nil, err
	}
	encodedData := base64.StdEncoding.EncodeToString(randomData)
	b.AddItem(InternalSharedSecretEnvName, encodedData)

	return b.GetObject(), nil
}

var _ reconciler.Reconciler = &InternalSharedSecretReconciler{}

type InternalSharedSecretReconciler struct {
	reconciler.GenericResourceReconciler[*InternalSharedSecretBuilder]
}

func NewInternalsharedSecretReconciler(
	client *client.Client,
	info reconciler.ClusterInfo,
) reconciler.Reconciler {
	name := getInternalSharedSecretName(info.GetClusterName())
	builder := &InternalSharedSecretBuilder{
		SecretBuilder: *builder.NewSecretBuilder(
			client,
			name,
			info.GetLabels(),
			info.GetAnnotations(),
		),
	}

	return &InternalSharedSecretReconciler{
		GenericResourceReconciler: *reconciler.NewGenericResourceReconciler(
			client,
			name,
			builder,
		),
	}
}

// Create a contains a random secret for trino internal communication
// If it does not exist, create it, otherwise do nothing
func (r *InternalSharedSecretReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if err := r.Client.Client.Get(
		ctx,
		ctrlclient.ObjectKey{Namespace: r.Client.GetOwnerNamespace(), Name: r.Name},
		&corev1.Secret{},
	); err != nil {
		if ctrlclient.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// Secret does not exist, create it
		return r.GenericResourceReconciler.Reconcile(ctx)
	}
	return ctrl.Result{}, nil
}
