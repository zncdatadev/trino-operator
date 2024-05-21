package common

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

var log = ctrl.Log.WithName("resourceFetcher")

type ResourceClient struct {
	Ctx       context.Context
	Client    client.Client
	Namespace string
}

func (r *ResourceClient) Get(obj client.Object) error {
	name := obj.GetName()
	kind := obj.GetObjectKind()
	if err := r.Client.Get(r.Ctx, client.ObjectKey{Namespace: r.Namespace, Name: name}, obj); err != nil {
		opt := []any{"ns", r.Namespace, "name", name, "kind", kind}
		if apierrors.IsNotFound(err) {
			log.Error(err, "Fetch resource NotFound", opt...)
		} else {
			log.Error(err, "Fetch resource occur some unknown err", opt...)
		}
		return err
	}
	return nil
}

type InstanceAttributes interface {
	RoleConfigSpec
	GetClusterConfig() any
	GetClusterOperation() *trinov1alpha1.ClusterOperationSpec
}

type TrinoInstance struct {
	Instance *trinov1alpha1.TrinoCluster
}

// GetClusterConfig implement InstanceAttributes interface
func (t *TrinoInstance) GetClusterConfig() any {
	return t.Instance.Spec.ClusterConfig
}

func (i *TrinoInstance) GetClusterOperation() *trinov1alpha1.ClusterOperationSpec {
	return i.Instance.Spec.ClusterOperation
}

func (t *TrinoInstance) GetRoleConfigSpec(role Role) (any, error) {
	switch role {
	case Coordinator:
		return t.Instance.Spec.Coordinator, nil
	case Worker:
		return t.Instance.Spec.Worker, nil
	default:
		return nil, fmt.Errorf("role %s not found", role)
	}
}
