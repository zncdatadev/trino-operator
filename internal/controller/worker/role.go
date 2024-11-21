package worker

import (
	"context"

	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/builder"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/util"
	corev1 "k8s.io/api/core/v1"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/controller/common"
)

var _ reconciler.RoleReconciler = &Reconciler{}

type Reconciler struct {
	reconciler.BaseRoleReconciler[*trinov1alpha1.WorkersSpec]

	ClusterConfig      *trinov1alpha1.ClusterConfigSpec
	Image              *util.Image
	CoordiantorSvcFqdn string
}

func NewReconciler(
	client *client.Client,
	clusterStopped bool,
	clusterConfig *trinov1alpha1.ClusterConfigSpec,
	roleInfo reconciler.RoleInfo,
	image *util.Image,
	coordiantorSvcFqdn string,
	spec *trinov1alpha1.WorkersSpec,
) *Reconciler {
	return &Reconciler{
		BaseRoleReconciler: *reconciler.NewBaseRoleReconciler(client, clusterStopped, roleInfo, spec),
		ClusterConfig:      clusterConfig,
		Image:              image,
		CoordiantorSvcFqdn: coordiantorSvcFqdn,
	}
}

func (r *Reconciler) RegisterResources(ctx context.Context) error {
	for name, roleGroup := range r.Spec.RoleGroups {
		mergedConfig, err := util.MergeObject(r.Spec.Config, roleGroup.Config)
		if err != nil {
			return err
		}

		mergedOverrides, err := util.MergeObject(r.Spec.OverridesSpec, roleGroup.OverridesSpec)
		if err != nil {
			return err
		}

		info := reconciler.RoleGroupInfo{RoleInfo: r.RoleInfo, RoleGroupName: name}

		reconcilers, err := r.registerResourceWithRoleGroup(info, mergedConfig, mergedOverrides, roleGroup.Replicas)
		if err != nil {
			return err
		}

		for _, reconciler := range reconcilers {
			r.AddResource(reconciler)
		}
	}

	return nil
}

func (r *Reconciler) registerResourceWithRoleGroup(
	info reconciler.RoleGroupInfo,
	roleGroupConfig *trinov1alpha1.ConfigSpec,
	overrideSpec *commonsv1alpha1.OverridesSpec,
	replicas *int32,
) ([]reconciler.Reconciler, error) {
	var reconcilers []reconciler.Reconciler

	ports := []corev1.ContainerPort{{Name: "http", ContainerPort: trinov1alpha1.HttpPort}}

	if r.ClusterConfig != nil && r.ClusterConfig.Tls != nil {
		ports = []corev1.ContainerPort{{Name: "https", ContainerPort: trinov1alpha1.HttpsPort}}
	}

	configMapReconciler := common.NewConfigReconciler(
		r.Client,
		r.CoordiantorSvcFqdn,
		r.ClusterConfig,
		roleGroupConfig,
		info,
	)

	reconcilers = append(reconcilers, configMapReconciler)

	serviceReconciler := reconciler.NewServiceReconciler(
		r.Client,
		info.GetFullName(),
		ports,
	)
	reconcilers = append(reconcilers, serviceReconciler)

	statefulSetReconciler, err := common.NewStatefulSetReconciler(
		r.Client,
		r.ClusterConfig,
		info,
		r.Image,
		r.ClusterStopped(),
		replicas,
		ports,
		overrideSpec,
		roleGroupConfig,
		func(o *builder.Options) {
			o.ClusterName = info.GetClusterName()
			o.RoleName = info.GetRoleName()
			o.RoleGroupName = info.GetGroupName()
			o.Labels = info.GetLabels()
			o.Annotations = info.GetAnnotations()
		},
	)
	if err != nil {
		return nil, err
	}

	reconcilers = append(reconcilers, statefulSetReconciler)

	return reconcilers, nil
}
