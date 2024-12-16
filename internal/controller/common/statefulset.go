package common

import (
	"path"
	"strings"

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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinosv1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/controller/common/authz"
)

var (
	TrinoConfigDir      = constants.KubedoopConfigDir
	TrinoConfigMountDir = constants.KubedoopConfigDirMount
	TrinoDataDir        = constants.KubedoopDataDir
	TrinoLogDir         = constants.KubedoopLogDir

	TrinoConfigVolumeName      = "config"
	TrinoDataVolumeName        = "data"
	TrinoLogVolumeName         = "log"
	TrinoServerTlsVolumeName   = "server-tls"
	TrinoInternalTlsVolumeName = "internal-tls"
	TrinoClientTlsVolumeName   = "client-tls"
)

func NewStatefulSetReconciler(
	client *client.Client,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	roleGroupInfo reconciler.RoleGroupInfo,
	image *util.Image,
	stopped bool,
	replicas *int32,
	ports []corev1.ContainerPort,
	overrides *commonsv1alpha1.OverridesSpec,
	roleGroupConfig *trinosv1alpha1.ConfigSpec,
	options ...builder.Option,
) (*reconciler.StatefulSet, error) {

	opts := &builder.Options{}

	for _, o := range options {
		o(opts)
	}

	var commonsRoleGroupConfig *commonsv1alpha1.RoleGroupConfigSpec

	if roleGroupConfig != nil {
		commonsRoleGroupConfig = roleGroupConfig.RoleGroupConfigSpec
	}

	builder := NewStatefulSetBuilder(
		client,
		roleGroupInfo.GetFullName(),
		replicas,
		image,
		clusterConfig,
		ports,
		overrides,
		commonsRoleGroupConfig,
		options...,
	)

	return reconciler.NewStatefulSet(
		client,
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
	ClusterName   string
	RoleName      string
	ports         []corev1.ContainerPort
}

func NewStatefulSetBuilder(
	client *client.Client,
	name string,
	replicas *int32,
	image *util.Image,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	ports []corev1.ContainerPort,
	overrides *commonsv1alpha1.OverridesSpec,
	roleGroupConfig *commonsv1alpha1.RoleGroupConfigSpec,
	options ...builder.Option,
) *StatefulSetBuilder {

	opts := &builder.Options{}
	for _, o := range options {
		o(opts)
	}

	return &StatefulSetBuilder{
		StatefulSet: *builder.NewStatefulSetBuilder(
			client,
			name,
			replicas,
			image,
			overrides,
			roleGroupConfig,
			options...,
		),
		ClusterConfig: clusterConfig,
		RoleName:      opts.RoleName,
		ClusterName:   opts.ClusterName,
		Image:         image,
		ports:         ports,
	}
}

func (b *StatefulSetBuilder) Build(ctx context.Context) (ctrlclient.Object, error) {
	b.AddVolumeClaimTemplates(b.getPvcTemplates())

	volumes, err := b.getVolumes(ctx)
	if err != nil {
		return nil, err
	}
	b.AddVolumes(volumes)

	container, err := b.getMainContainer(ctx)
	if err != nil {
		return nil, err
	}
	b.AddContainer(container)
	obj, err := b.GetObject()
	if err != nil {
		return nil, err
	}
	if b.ClusterConfig != nil && b.ClusterConfig.VectorAggregatorConfigMapName != "" {
		decorator := builder.NewVectorDecorator(
			obj,
			b.Image,
			TrinoLogVolumeName,
			TrinoConfigVolumeName,
			b.ClusterConfig.VectorAggregatorConfigMapName)
		if err := decorator.Decorate(); err != nil {
			return nil, err
		}
	}
	return obj, nil
}

func (b *StatefulSetBuilder) enabledTls() bool {
	return b.ClusterConfig != nil && b.ClusterConfig.Tls != nil
}

func (b *StatefulSetBuilder) getMainContainer(ctx context.Context) (*corev1.Container, error) {
	container := builder.NewContainer(b.RoleName, b.Image)
	container.SetCommand([]string{"sh", "-c"})

	args, err := b.getMainContainerArgs(ctx)
	if err != nil {
		return nil, err
	}
	container.SetArgs(args)

	volumeMounts, err := b.getMainContainerVolumeMounts(ctx)
	if err != nil {
		return nil, err
	}
	container.AddVolumeMounts(volumeMounts)
	if b.enabledTls() {
		container.AddEnvFromSecret(getInternalSharedSecretName(b.ClusterName))
	}
	envVars, err := b.getMainContainerEnvVars(ctx)
	if err != nil {
		return nil, err
	}
	container.AddEnvVars(envVars)
	container.AddPorts(b.ports)

	portName := "http"
	schema := corev1.URISchemeHTTP
	if b.enabledTls() {
		portName = "https"
		schema = corev1.URISchemeHTTPS
	}
	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/info",
				Port:   intstr.FromString(portName),
				Scheme: schema,
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		FailureThreshold:    6,
		SuccessThreshold:    1,
	}
	container.SetLivenessProbe(probe)
	container.SetReadinessProbe(probe)
	return container.Build(), nil
}

func (b *StatefulSetBuilder) getMainContainerEnvVars(ctx context.Context) ([]corev1.EnvVar, error) {
	envVars := make([]corev1.EnvVar, 0)
	if b.ClusterConfig != nil && b.ClusterConfig.Authentication != nil {
		auth, err := authz.NewAuthentication(ctx, b.Client, b.ClusterConfig.Authentication)
		if err != nil {
			return nil, err
		}
		envVars = append(envVars, auth.GetEnvVars()...)
	}

	return envVars, nil
}

func (b *StatefulSetBuilder) getMainContainerArgs(ctx context.Context) ([]string, error) {
	// TODO: Add s3 tls verification, add s3 truststore to client truststore
	authCommands := ""
	if b.ClusterConfig != nil && b.ClusterConfig.Authentication != nil {
		auth, err := authz.NewAuthentication(ctx, b.Client, b.ClusterConfig.Authentication)
		if err != nil {
			return nil, err
		}

		authCommands = strings.Join(auth.GetCommands(), "\n")
	}

	arg := `
set -ex
mkdir -p ` + TrinoConfigDir + `
cp ` + path.Join(TrinoConfigMountDir, "*") + ` ` + TrinoConfigDir + `

# TODO: remove this in futrue ,Move catalog files to catalog directory
mkdir -p ` + TrinoConfigDir + "catalog" + `
for f in ` + path.Join(TrinoConfigDir, "catalog-*") + `; do
  if [ -f "$f" ]; then
	echo "Moving $f to catalog directory"
    filename=$(basename "$f")
    newname=$(echo "$filename" | sed 's/^catalog-//')
    mv "$f" ` + path.Join(TrinoConfigDir, "catalog", "$newname") + `
  fi
done

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

keytool \
	-importkeystore \
	-srckeystore /etc/pki/java/cacerts \
	-srcstoretype JKS \
	-srcstorepass ` + DefaultTlsPassphrase + `\
	-destkeystore ` + path.Join(ClientTlsPath, "truststore.p12") + `\
	-deststoretype PKCS12 \
	-deststorepass ` + DefaultTlsPassphrase + `\
	-noprompt

` + authCommands + `

bin/launcher run --etc-dir ` + TrinoConfigDir + ` --data-dir ` + TrinoDataDir + `
wait_for_termination $!
mkdir -p /kubedoop/log/_vector && touch /kubedoop/log/_vector/shutdown
`
	return []string{util.IndentTab4Spaces(arg)}, nil
}

func (b *StatefulSetBuilder) getMainContainerVolumeMounts(ctx context.Context) ([]corev1.VolumeMount, error) {
	volumes := []corev1.VolumeMount{
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
		{
			Name:      TrinoClientTlsVolumeName,
			MountPath: ClientTlsPath,
		},
	}

	if b.enabledTls() {
		volumes = append(volumes, corev1.VolumeMount{
			Name:      TrinoServerTlsVolumeName,
			MountPath: ServerTlsMountPath,
		},
			corev1.VolumeMount{
				Name:      TrinoInternalTlsVolumeName,
				MountPath: InternalTlsMountPath,
			},
		)
	}

	if b.ClusterConfig != nil && b.ClusterConfig.Authentication != nil {
		auth, err := authz.NewAuthentication(ctx, b.Client, b.ClusterConfig.Authentication)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, auth.GetVolumeMounts()...)
	}

	return volumes, nil
}

func (b *StatefulSetBuilder) getVolumes(ctx context.Context) ([]corev1.Volume, error) {
	volumes := []corev1.Volume{
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
		{
			Name: TrinoClientTlsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: ptr.To(resource.MustParse("1Mi")),
				},
			},
		},
	}

	if b.enabledTls() {
		secretClassName := b.ClusterConfig.Tls.ServerSecretClass
		if secretClassName == "" {
			secretClassName = "tls"
		}
		volumes = append(volumes, buildTlsVolume(TrinoServerTlsVolumeName, secretClassName), buildTlsVolume(TrinoInternalTlsVolumeName, secretClassName))
	}

	if b.ClusterConfig != nil && b.ClusterConfig.Authentication != nil {
		auth, err := authz.NewAuthentication(ctx, b.Client, b.ClusterConfig.Authentication)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, auth.GetVolumes()...)
	}

	return volumes, nil
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

func buildTlsVolume(name string, secretClassName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Ephemeral: &corev1.EphemeralVolumeSource{
				VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							constants.AnnotationSecretsClass:          secretClassName,
							constants.AnnotationSecretsScope:          strings.Join([]string{string(constants.PodScope), string(constants.NodeScope)}, ","),
							constants.AnnotationSecretsFormat:         string(constants.TLSP12),
							constants.AnnotationSecretsPKCS12Password: DefaultTlsPassphrase,
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
						StorageClassName: constants.SecretStorageClassPtr(),
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Mi"),
							},
						},
					},
				},
			},
		},
	}
}
