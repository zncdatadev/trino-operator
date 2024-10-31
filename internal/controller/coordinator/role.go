package coordinator

import (
	"context"
	"time"

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
	reconciler.BaseRoleReconciler[*trinov1alpha1.CoordinatorsSpec]

	ClusterConfig      *trinov1alpha1.ClusterConfigSpec
	Image              *util.Image
	CoordiantorSvcFqdn string
}

func NewWorkerReconciler(
	client *client.Client,
	clusterStopped bool,
	clusterConfig *trinov1alpha1.ClusterConfigSpec,
	roleInfo reconciler.RoleInfo,
	image *util.Image,
	coordiantorSvcFqdn string,
	spec *trinov1alpha1.CoordinatorsSpec,
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
		mergedRoleGroup := r.MergeRoleGroupSpec(roleGroup)

		info := reconciler.RoleGroupInfo{RoleInfo: r.RoleInfo, RoleGroupName: name}

		reconcilers, err := r.registerResourceWithRoleGroup(ctx, info, mergedRoleGroup)
		if err != nil {
			return err
		}

		for _, reconciler := range reconcilers {
			r.AddResource(reconciler)
		}
	}

	return nil
}

func (r *Reconciler) registerResourceWithRoleGroup(_ context.Context, info reconciler.RoleGroupInfo, roleGroupSpec any) ([]reconciler.Reconciler, error) {
	spec := roleGroupSpec.(*trinov1alpha1.RoleGroupSpec)
	var reconcilers []reconciler.Reconciler

	ports := []corev1.ContainerPort{{Name: "http", ContainerPort: trinov1alpha1.HttpPort}}

	if r.ClusterConfig != nil && r.ClusterConfig.Tls != nil {
		ports = []corev1.ContainerPort{{Name: "https", ContainerPort: trinov1alpha1.HttpsPort}}
	}

	options := builder.WorkloadOptions{
		Options: builder.Options{
			ClusterName:   info.GetClusterName(),
			RoleName:      info.GetRoleName(),
			RoleGroupName: info.RoleGroupName,
			Labels:        info.GetLabels(),
			Annotations:   info.GetAnnotations(),
		},
		CommandOverrides: spec.CliOverrides,
		EnvOverrides:     spec.EnvOverrides,
	}

	if spec.Config != nil {
		if spec.Config.GracefulShutdownTimeout != nil {
			gracefulShutdownTimeout, err := time.ParseDuration(*spec.Config.GracefulShutdownTimeout)
			if err != nil {
				return nil, err
			}
			options.TerminationGracePeriod = &gracefulShutdownTimeout
		}

		options.Resource = spec.Config.Resources
		options.Affinity = spec.Config.Affinity
	}

	configMapReconciler := common.NewConfigReconciler(
		r.Client,
		r.CoordiantorSvcFqdn,
		r.ClusterConfig,
		spec.Config,
		info,
	)

	reconcilers = append(reconcilers, configMapReconciler)

	serviceReconciler := reconciler.NewServiceReconciler(
		r.Client,
		info.GetFullName(),
		ports,
		func(sbo *builder.ServiceBuilderOption) {
			sbo.Labels = info.GetLabels()
			sbo.Annotations = info.GetAnnotations()
			sbo.ClusterName = info.GetClusterName()
			sbo.RoleName = info.GetRoleName()
			sbo.RoleGroupName = info.RoleGroupName
		},
	)
	reconcilers = append(reconcilers, serviceReconciler)

	statefulSetReconciler, err := common.NewStatefulSetReconciler(
		r.Client,
		r.ClusterConfig,
		info,
		r.Image,
		r.ClusterStopped,
		spec.Replicas,
		ports,
		options,
	)
	if err != nil {
		return nil, err
	}

	reconcilers = append(reconcilers, statefulSetReconciler)

	return reconcilers, nil
}
