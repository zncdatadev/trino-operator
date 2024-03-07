package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	stackv1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	"github.com/zncdata-labs/trino-operator/internal/common"
	"github.com/zncdata-labs/trino-operator/internal/controller/coordinator"
	"github.com/zncdata-labs/trino-operator/internal/controller/worker"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TrinoInstance struct {
	instance *stackv1alpha1.TrinoCluster
}

// GetClusterConfig implement InstanceAttributes interface
func (t *TrinoInstance) GetClusterConfig() any {
	return t.instance.Spec.ClusterConfig
}

func (t *TrinoInstance) GetRoleConfigSpec(role common.Role) (any, error) {
	switch role {
	case common.Coordinator:
		return t.instance.Spec.Coordinator, nil
	case common.Worker:
		return t.instance.Spec.Worker, nil
	default:
		return nil, fmt.Errorf("role %s not found", role)
	}
}

type ClusterReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	cr     *stackv1alpha1.TrinoCluster
	Log    logr.Logger

	roleReconcilers     []common.RoleReconciler
	resourceReconcilers []common.ResourceReconciler
}

func NewClusterReconciler(client client.Client, scheme *runtime.Scheme, cr *stackv1alpha1.TrinoCluster) *ClusterReconciler {
	c := &ClusterReconciler{
		client: client,
		scheme: scheme,
		cr:     cr,
	}
	c.RegisterRole()
	c.RegisterResource()
	return c
}

// RegisterRole register role reconciler
func (c *ClusterReconciler) RegisterRole() {
	coordinatorRole := coordinator.NewRoleCoordinator(c.scheme, c.cr, c.client, c.Log)
	workerRole := worker.NewRoleWorker(c.scheme, c.cr, c.client, c.Log)
	c.roleReconcilers = []common.RoleReconciler{coordinatorRole, workerRole}
}

func (c *ClusterReconciler) RegisterResource() {
	cm := NewClusterConfigMap(c.scheme, c.cr, c.client, "", c.cr.Labels, nil)
	c.resourceReconcilers = []common.ResourceReconciler{cm}
}

func (c *ClusterReconciler) ReconcileCluster(ctx context.Context) (ctrl.Result, error) {
	if len(c.resourceReconcilers) > 0 {
		res, err := common.ReconcilerDoHandler(ctx, c.resourceReconcilers)
		if err != nil {
			return ctrl.Result{}, err
		}
		if res.RequeueAfter > 0 {
			return res, nil
		}
	}

	for _, r := range c.roleReconcilers {
		res, err := r.ReconcileRole(ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
		if res.RequeueAfter > 0 {
			return res, nil
		}
	}
	return ctrl.Result{}, nil
}
