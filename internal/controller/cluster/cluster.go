package cluster

import (
	"context"
	"strings"

	"github.com/zncdatadev/operator-go/pkg/builder"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/util"
	corev1 "k8s.io/api/core/v1"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/controller/common"
	"github.com/zncdatadev/trino-operator/internal/controller/coordinator"
	"github.com/zncdatadev/trino-operator/internal/controller/worker"
	"github.com/zncdatadev/trino-operator/internal/util/version"
)

var _ reconciler.Reconciler = &Reconciler{}

type Reconciler struct {
	reconciler.BaseCluster[*trinov1alpha1.TrinoClusterSpec]

	ClusterConfig *trinov1alpha1.ClusterConfigSpec
}

func NewClusterReconciler(
	client *client.Client,
	clusterInfo reconciler.ClusterInfo,
	spec *trinov1alpha1.TrinoClusterSpec,
) *Reconciler {
	return &Reconciler{
		BaseCluster:   *reconciler.NewBaseCluster(client, clusterInfo, spec.ClusterOperation, spec),
		ClusterConfig: spec.ClusterConfig,
	}
}

func (r *Reconciler) GetImage() *util.Image {
	image := util.NewImage(
		trinov1alpha1.DefaultProductName,
		version.BuildVersion,
		trinov1alpha1.DefaultProductVersion,
		func(options *util.ImageOptions) {
			options.Custom = r.Spec.Image.Custom
			options.Repo = r.Spec.Image.Repo
			options.PullPolicy = r.Spec.Image.PullPolicy
		},
	)

	if r.Spec.Image.KubedoopVersion != "" {
		image.KubedoopVersion = r.Spec.Image.KubedoopVersion
	}

	return image
}

func (r *Reconciler) getCoordinatorSvcFqdn() string {
	fqdns := make([]string, 0)
	coordinator := r.Spec.Coordinators

	if coordinator.RoleGroups != nil {
		// "coordinator-"+name+"."+r.Client.GetOwnerNamespace()+".svc.cluster.local"
		for name := range coordinator.RoleGroups {
			roleGroupInfo := reconciler.RoleGroupInfo{RoleInfo: reconciler.RoleInfo{ClusterInfo: r.ClusterInfo, RoleName: "coordinator"}, RoleGroupName: name}
			fqdns = append(fqdns, strings.Join([]string{roleGroupInfo.GetFullName(), r.Client.GetOwnerNamespace(), "svc.cluster.local"}, "."))
		}
	}

	// Ensure there is at least one coordinator
	if len(fqdns) > 0 {
		return fqdns[0]
	}
	return ""
}

func (r *Reconciler) RegisterResources(ctx context.Context) error {
	listenerClass := trinov1alpha1.DefaultListenerClass
	var enabledTls bool
	containerPort := corev1.ContainerPort{Name: "http", ContainerPort: trinov1alpha1.HttpPort}
	if r.ClusterConfig != nil {
		listenerClass = r.ClusterConfig.ListenerClass
		enabledTls = r.ClusterConfig.Tls != nil
	}

	coordinatorSvcFqdn := r.getCoordinatorSvcFqdn()
	coordinatorRoleInfo := reconciler.RoleInfo{ClusterInfo: r.ClusterInfo, RoleName: "coordinator"}
	coordinatorReconciler := coordinator.NewWorkerReconciler(
		r.Client,
		r.IsStopped(),
		r.ClusterConfig,
		coordinatorRoleInfo,
		r.GetImage(),
		coordinatorSvcFqdn,
		r.Spec.Coordinators,
	)
	if err := coordinatorReconciler.RegisterResources(ctx); err != nil {
		return err
	}
	r.AddResource(coordinatorReconciler)

	workerReconciler := worker.NewReconciler(
		r.Client,
		r.IsStopped(),
		r.ClusterConfig,
		reconciler.RoleInfo{ClusterInfo: r.ClusterInfo, RoleName: "worker"},
		r.GetImage(),
		coordinatorSvcFqdn,
		r.Spec.Workers,
	)
	if err := workerReconciler.RegisterResources(ctx); err != nil {
		return err
	}
	r.AddResource(workerReconciler)

	if enabledTls {
		secretReconciler := common.NewInternalsharedSecretReconciler(
			r.Client,
			r.ClusterInfo,
		)
		r.AddResource(secretReconciler)

		containerPort = corev1.ContainerPort{Name: "https", ContainerPort: trinov1alpha1.HttpsPort}
	}

	serviceReconciler := reconciler.NewServiceReconciler(
		r.Client,
		coordinatorRoleInfo.GetFullName(),
		[]corev1.ContainerPort{containerPort},
		func(o *builder.ServiceBuilderOptions) {
			o.Labels = coordinatorRoleInfo.GetLabels()
			o.Annotations = coordinatorRoleInfo.GetAnnotations()
			o.ClusterName = r.ClusterInfo.GetClusterName()
			o.RoleName = coordinatorRoleInfo.RoleName
			o.ListenerClass = listenerClass
			o.MatchingLabels = coordinatorRoleInfo.GetLabels()
		},
	)
	r.AddResource(serviceReconciler)

	return nil
}
