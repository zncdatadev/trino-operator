package worker

import (
	"context"
	"github.com/go-logr/logr"
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	"github.com/zncdatadev/trino-operator/internal/util"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// role Worker reconciler

type Role struct {
	common.BaseRoleReconciler[*trinov1alpha1.TrinoCluster]
}

// NewRoleWorker NewRoleMaster new roleMaster
func NewRoleWorker(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	log logr.Logger) *Role {
	r := &Role{
		BaseRoleReconciler: common.BaseRoleReconciler[*trinov1alpha1.TrinoCluster]{
			Scheme:   scheme,
			Instance: instance,
			Client:   client,
			Log:      log,
			Role:     common.Worker,
		},
	}
	r.Labels = r.MergeLabels()
	return r
}

func (r *Role) RoleName() common.Role {
	return common.Worker
}

func (r *Role) MergeLabels() map[string]string {
	return r.GetLabels()
}

func (r *Role) ReconcileRole(ctx context.Context) (ctrl.Result, error) {
	roleCfg := r.Instance.Spec.Worker
	// role pdb
	if roleCfg.Config != nil && roleCfg.Config.PodDisruptionBudget != nil {
		pdb := common.NewReconcilePDB(r.Client, r.Scheme, r.Instance, r.Labels, string(r.RoleName()),
			roleCfg.PodDisruptionBudget)
		res, err := pdb.ReconcileResource(ctx, common.NewSingleResourceBuilder(pdb))
		if err != nil {
			return ctrl.Result{}, err
		}
		if res.RequeueAfter > 0 {
			return res, nil
		}
	}
	// reconciler groups
	for name := range roleCfg.RoleGroups {
		groupReconciler := NewRoleGroupReconciler(r.Scheme, r.Instance, r.Client, name, r.Labels, r.Log)
		res, err := groupReconciler.ReconcileGroup(ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
		if res.RequeueAfter > 0 {
			return res, nil
		}
	}
	return ctrl.Result{}, nil
}

// RoleGroup master role group reconcile
type RoleGroup struct {
	common.BaseRoleGroupReconciler[*trinov1alpha1.TrinoCluster]
}

func NewRoleGroupReconciler(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	roleLabels map[string]string,
	log logr.Logger) *RoleGroup {
	r := &RoleGroup{
		BaseRoleGroupReconciler: common.BaseRoleGroupReconciler[*trinov1alpha1.TrinoCluster]{
			Scheme:     scheme,
			Instance:   instance,
			Client:     client,
			GroupName:  groupName,
			RoleLabels: roleLabels,
			Log:        log,
		},
	}
	r.RegisterResource()
	return r
}

func (m *RoleGroup) RegisterResource() {
	cfg := m.MergeGroupConfigSpec()
	lables := m.MergeLabels(cfg)
	mergedCfg := cfg.(*trinov1alpha1.RoleGroupSpec)
	pdbSpec := mergedCfg.Config.PodDisruptionBudget
	logDataBuilder := &LogDataBuilder{cfg: mergedCfg}

	pdb := common.NewReconcilePDB(m.Client, m.Scheme, m.Instance, lables, m.GroupName, pdbSpec)
	cm := NewConfigMap(m.Scheme, m.Instance, m.Client, m.GroupName, lables, mergedCfg)
	log := NewWorkerLogging(m.Scheme, m.Instance, m.Client, m.GroupName, lables, mergedCfg, logDataBuilder, common.Worker)
	dep := NewDeployment(m.Scheme, m.Instance, m.Client, m.GroupName, lables, mergedCfg, mergedCfg.Replicas)
	m.Reconcilers = []common.ResourceReconciler{pdb, cm, log, dep}
}

func (m *RoleGroup) MergeGroupConfigSpec() any {
	originMasterCfg := m.Instance.Spec.Worker.RoleGroups[m.GroupName]
	instance := m.Instance
	// Merge the role into the role group.
	// if the role group has a config, and role group not has a config, will
	// merge the role's config into the role group's config.
	return mergeConfig(instance.Spec.Worker, originMasterCfg)
}

func (m *RoleGroup) MergeLabels(mergedCfg any) map[string]string {
	mergedMasterCfg := mergedCfg.(*trinov1alpha1.RoleGroupSpec)
	roleLabels := m.RoleLabels
	mergeLabels := make(util.Map)
	mergeLabels.MapMerge(roleLabels, true)
	mergeLabels.MapMerge(mergedMasterCfg.Config.NodeSelector, true)
	mergeLabels["app.kubernetes.io/instance"] = strings.ToLower(m.GroupName)
	return mergeLabels
}

// mergeConfig merge the role's config into the role group's config
func mergeConfig(masterRole *trinov1alpha1.WorkerSpec,
	group *trinov1alpha1.RoleGroupSpec) *trinov1alpha1.RoleGroupSpec {
	copiedRoleGroup := group.DeepCopy()
	// Merge the role into the role group.
	// if the role group has a config, and role group not has a config, will
	// merge the role's config into the role group's config.
	common.MergeObjects(copiedRoleGroup, masterRole, []string{"RoleGroups"})

	// merge the role's config into the role group's config
	if masterRole.Config != nil && copiedRoleGroup.Config != nil {
		common.MergeObjects(copiedRoleGroup.Config, masterRole.Config, []string{})
	}
	return copiedRoleGroup
}
