package common

import (
	"path"

	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/builder"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/constants"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/util"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinosv1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

var (
	TrinoConfigDir      = constants.KubedoopConfigDir
	TrinoConfigMountDir = constants.KubedoopConfigDirMount
	TrinoDataDir        = constants.KubedoopDataDir
	TrinoLogDir         = constants.KubedoopLogDir

	TrinoConfigVolumeName = "config"
	TrinoDataVolumeName   = "data"
	TrinoLogVolumeName    = "log"

	HttpPort int32 = 8080
)

func NewStatefulSetReconciler(
	client *client.Client,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	roleGroupInfo reconciler.RoleGroupInfo,
	image *util.Image,
	stopped bool,
	replicas *int32,
	options builder.WorkloadOptions,
) (*reconciler.StatefulSet, error) {
	builder := NewStatefulSetBuilder(
		client,
		roleGroupInfo.GetFullName(),
		replicas,
		image,
		clusterConfig,
		options,
	)

	return reconciler.NewStatefulSet(
		client,
		roleGroupInfo.GetFullName(),
		builder,
		stopped,
	), nil
}

var _ builder.StatefulSetBuilder = &StatefulSetBuilder{}

type StatefulSetBuilder struct {
	builder.StatefulSet

	ClusterConfig *trinosv1alpha1.ClusterConfigSpec
	Resource      *commonsv1alpha1.ResourcesSpec
	Image         *util.Image
	RoleName      string
}

func NewStatefulSetBuilder(
	client *client.Client,
	name string,
	replicas *int32,
	image *util.Image,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	options builder.WorkloadOptions,
) *StatefulSetBuilder {
	return &StatefulSetBuilder{
		StatefulSet: *builder.NewStatefulSetBuilder(
			client,
			name,
			replicas,
			image,
			options,
		),
		ClusterConfig: clusterConfig,
		RoleName:      options.RoleName,
		Image:         image,
	}
}

func (b *StatefulSetBuilder) Build(ctx context.Context) (ctrlclient.Object, error) {
	b.AddVolumeClaimTemplates(b.getPvcTemplates())
	b.AddVolumes(b.getVolumes())
	b.AddContainer(b.getMainContainer())
	return b.GetObject()
}

func (b *StatefulSetBuilder) getMainContainer() *corev1.Container {
	container := builder.NewContainer(b.RoleName, b.Image)
	container.SetCommand([]string{"sh", "-c"})
	container.SetArgs(b.getMainContainerArgs())
	container.AddVolumeMounts(b.getMainContainerVolumeMounts())
	container.AddPort(corev1.ContainerPort{
		Name:          "http",
		ContainerPort: int32(HttpPort),
	})

	return container.Build()
}

func (b *StatefulSetBuilder) getMainContainerArgs() []string {
	arg := `
set -ex
mkdir -p ` + TrinoConfigDir + `
cp ` + path.Join(TrinoConfigMountDir, "*") + ` ` + TrinoConfigDir + `


prepare_signal_handlers()
{
    unset term_child_pid
    unset term_kill_needed
    trap 'handle_term_signal' TERM
}

handle_term_signal()
{
    if [ "${term_child_pid}" ]; then
        kill -TERM "${term_child_pid}" 2>/dev/null
    else
        term_kill_needed="yes"
    fi
}

wait_for_termination()
{
    set +e
    term_child_pid=$1
    if [[ -v term_kill_needed ]]; then
        kill -TERM "${term_child_pid}" 2>/dev/null
    fi
    wait ${term_child_pid} 2>/dev/null
    trap - TERM
    wait ${term_child_pid} 2>/dev/null
    set -e
} 

rm -f /kubedoop/log/_vector/shutdown
prepare_signal_handlers

bin/launcher run --etc-dir ` + TrinoConfigDir + ` --data-dir ` + TrinoDataDir + `
wait_for_termination $!
mkdir -p /kubedoop/log/_vector && touch /kubedoop/log/_vector/shutdown
`
	return []string{util.IndentTab4Spaces(arg)}
}

func (b *StatefulSetBuilder) getMainContainerVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      TrinoConfigVolumeName,
			MountPath: TrinoConfigMountDir,
		},
		{
			Name:      TrinoDataVolumeName,
			MountPath: TrinoDataDir,
		},
		{
			Name:      TrinoLogVolumeName,
			MountPath: TrinoLogDir,
		},
	}
}

// func (b *StatefulSetBuilder) getVectorContainer() *corev1.Container {
// 	panic("implement me")
// }

func (b *StatefulSetBuilder) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: TrinoConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: b.GetName(),
					},
				},
			},
		},
		{
			Name: TrinoLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: ptr.To(resource.MustParse("1Gi")),
				},
			},
		},
	}
}

func (b *StatefulSetBuilder) getDataStorageSize() resource.Quantity {
	if b.Resource != nil && b.Resource.Storage != nil && !b.Resource.Storage.Capacity.IsZero() {
		return b.Resource.Storage.Capacity
	}
	return resource.MustParse("1Gi")
}

func (b *StatefulSetBuilder) getPvcTemplates() []corev1.PersistentVolumeClaim {
	return []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: TrinoDataVolumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: b.getDataStorageSize(),
					},
				},
			},
		},
	}
}
