package controller

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/builder"
	opgoconfig "github.com/zncdatadev/operator-go/pkg/config"
	"github.com/zncdatadev/operator-go/pkg/constant"
	"github.com/zncdatadev/operator-go/pkg/productlogging"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/security"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/controller/common/authz"
	"github.com/zncdatadev/trino-operator/internal/util/version"
)

const (
	LabelDomain = "trino.kubedoop.dev"

	configVolumeName      = "config"
	dataVolumeName        = "data"
	logVolumeName         = "log"
	serverTLSVolumeName   = "server-tls"
	internalTLSVolumeName = "internal-tls"
	clientTLSVolumeName   = "client-tls"

	internalSharedSecretEnvName = "INTERNAL_SHARED_SECRET"
	defaultTLSPassphrase        = "changeit"
)

var (
	serverTLSMountPath   = path.Join(constant.KubedoopTlsDir, "server")
	internalTLSMountPath = path.Join(constant.KubedoopTlsDir, "internal")
	clientTLSMountPath   = path.Join(constant.KubedoopTlsDir, "client")
)

var (
	_ reconciler.RoleGroupHandler[*trinov1alpha1.TrinoCluster]  = &TrinoRoleGroupHandler{}
	_ reconciler.RoleProvider[*trinov1alpha1.TrinoCluster]      = &TrinoRoleGroupHandler{}
	_ reconciler.RoleGroupResolver[*trinov1alpha1.TrinoCluster] = &TrinoRoleGroupHandler{}
	_ reconciler.VectorAggregatorProvider                       = (*trinov1alpha1.TrinoCluster)(nil)
)

// RBAC used by the GenericReconciler, the cluster-resource extension and authentication lookup.
//
// +kubebuilder:rbac:groups=trino.kubedoop.dev,resources=trinoclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=trino.kubedoop.dev,resources=trinoclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=trino.kubedoop.dev,resources=trinoclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=trino.kubedoop.dev,resources=trinocatalogs,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;services;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=authentication.kubedoop.dev,resources=authenticationclasses,verbs=get;list;watch

// TrinoRoleGroupHandler maps the standalone Trino API onto the operator-go Gen 3 resource model.
// Product-specific configuration, authentication and secret volumes are added around the base
// handler; naming, lifecycle, overrides, status and cleanup remain framework-owned.
type TrinoRoleGroupHandler struct {
	*reconciler.BaseRoleGroupHandler[*trinov1alpha1.TrinoCluster]
}

func ImageDefaults() commonsv1alpha1.ImageSpec {
	return commonsv1alpha1.ImageSpec{
		Repo:            trinov1alpha1.DefaultRepository,
		ProductVersion:  trinov1alpha1.DefaultProductVersion,
		KubedoopVersion: version.BuildVersion,
	}
}

func NewTrinoRoleGroupHandler(scheme *runtime.Scheme) *TrinoRoleGroupHandler {
	base := reconciler.NewBaseRoleGroupHandler[*trinov1alpha1.TrinoCluster](scheme)
	base.ConfigGenerator = opgoconfig.NewMultiFormatConfigGenerator()
	base.ConfigGenerator.RegisterDefaultFormats()
	base.ConfigGenerator.RegisterFormat(jvmFileName, jvmConfigAdapter{})
	base.ConfigMountPath = constant.KubedoopConfigDirMount
	base.LabelDomain = LabelDomain
	return &TrinoRoleGroupHandler{BaseRoleGroupHandler: base}
}

func (h *TrinoRoleGroupHandler) DeclareRoles(
	_ context.Context,
	_ client.Client,
	cr *trinov1alpha1.TrinoCluster,
) (reconciler.RoleCatalog, error) {
	portName := trinov1alpha1.HttpPortName
	port := trinov1alpha1.HttpPort
	scheme := corev1.URISchemeHTTP
	if tlsEnabled(cr) {
		portName = trinov1alpha1.HttpsPortName
		port = trinov1alpha1.HttpsPort
		scheme = corev1.URISchemeHTTPS
	}

	declaration := func(roleName string) reconciler.RoleDeclaration {
		return reconciler.RoleDeclaration{
			MainContainerName: roleName,
			ContainerPorts: []corev1.ContainerPort{
				{Name: portName, ContainerPort: port, Protocol: corev1.ProtocolTCP},
				{Name: trinov1alpha1.MetricsPortName, ContainerPort: trinov1alpha1.MetricsPort, Protocol: corev1.ProtocolTCP},
			},
			ServicePorts: []corev1.ServicePort{{
				Name:       portName,
				Port:       port,
				TargetPort: intstr.FromString(portName),
				Protocol:   corev1.ProtocolTCP,
			}},
			Command:        []string{"sh", "-c"},
			ReadinessProbe: trinoProbe(portName, scheme),
			LivenessProbe:  trinoProbe(portName, scheme),
			DataVolume: &reconciler.DataVolume{
				Name:      dataVolumeName,
				MountPath: constant.KubedoopDataDir,
			},
			LogProducers: []productlogging.ContainerLogging{{
				Container:   roleName,
				LogDirName:  trinov1alpha1.DefaultProductName,
				LogFileName: roleName + ".airlift.json",
			}},
			LogVolumeSize:  defaultStorageCapacity,
			ConfigDefaults: roleConfigDefaults(cr),
		}
	}

	return reconciler.RoleCatalog{
		trinov1alpha1.TrinoCoordinatorRoleName: declaration(trinov1alpha1.TrinoCoordinatorRoleName),
		trinov1alpha1.TrinoWorkerRoleName:      declaration(trinov1alpha1.TrinoWorkerRoleName),
	}, nil
}

func (h *TrinoRoleGroupHandler) ResolveRoleGroup(
	ctx context.Context,
	k8sClient client.Client,
	cr *trinov1alpha1.TrinoCluster,
	buildCtx *reconciler.RoleGroupBuildContext,
) (*reconciler.Contribution, error) {
	authentication, err := resolveAuthentication(ctx, k8sClient, cr)
	if err != nil {
		return nil, err
	}

	authProperties := map[string]string{}
	if authentication != nil {
		authProperties = authentication.GetConfigProperties()
	}

	contribution := &reconciler.Contribution{
		ConfigOverrides: trinoConfigFiles(cr, buildCtx, authProperties),
	}

	var env []corev1.EnvVar
	var envFrom []corev1.EnvFromSource
	if authentication != nil {
		env = authentication.GetEnvVars()
	}
	if tlsEnabled(cr) {
		envFrom = []corev1.EnvFromSource{{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: internalSharedSecretName(cr.Name)},
			},
		}}
	}
	if len(env) > 0 || len(envFrom) > 0 {
		contribution.PodOverrides = &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:    buildCtx.RoleName,
					Env:     env,
					EnvFrom: envFrom,
				}},
			},
		}
	}

	return contribution, nil
}

func (h *TrinoRoleGroupHandler) BuildResources(
	ctx context.Context,
	k8sClient client.Client,
	cr *trinov1alpha1.TrinoCluster,
	buildCtx *reconciler.RoleGroupBuildContext,
) (*reconciler.RoleGroupResources, error) {
	authentication, err := resolveAuthentication(ctx, k8sClient, cr)
	if err != nil {
		return nil, err
	}

	provider := &staticVolumeProvider{}
	provider.add(clientTLSVolume(), corev1.VolumeMount{Name: clientTLSVolumeName, MountPath: clientTLSMountPath})
	if !vectorEnabled(buildCtx) {
		provider.add(logVolume(), corev1.VolumeMount{Name: logVolumeName, MountPath: constant.KubedoopLogDir})
	}
	if tlsEnabled(cr) {
		serverClass, internalClass := tlsSecretClasses(cr)
		serverServiceNames := []string{
			buildCtx.ResourceName,
			buildCtx.ResourceName + "-headless",
		}
		if buildCtx.RoleName == trinov1alpha1.TrinoCoordinatorRoleName {
			serverServiceNames = append(
				serverServiceNames,
				cr.Name+"-"+trinov1alpha1.TrinoCoordinatorRoleName,
			)
		}
		provider.add(
			secretTLSVolume(serverTLSVolumeName, serverClass, serverServiceNames...),
			corev1.VolumeMount{Name: serverTLSVolumeName, MountPath: serverTLSMountPath},
		)
		provider.add(
			secretTLSVolume(internalTLSVolumeName, internalClass),
			corev1.VolumeMount{Name: internalTLSVolumeName, MountPath: internalTLSMountPath},
		)
	}
	if authentication != nil {
		provider.volumes = append(provider.volumes, authentication.GetVolumes()...)
		provider.mounts = append(provider.mounts, authentication.GetVolumeMounts()...)
	}
	buildCtx.VolumeProviders = append(buildCtx.VolumeProviders, provider)

	resources, err := h.BaseRoleGroupHandler.BuildResources(ctx, k8sClient, cr, buildCtx)
	if err != nil {
		return nil, err
	}

	mainContainer := findContainer(resources.StatefulSet.Spec.Template.Spec.Containers, buildCtx.RoleName)
	if mainContainer != nil && !podOverrideOwnsEntrypoint(buildCtx, buildCtx.RoleName) {
		if buildCtx.MergedConfig != nil && len(buildCtx.MergedConfig.CliArgs) > 0 {
			// operator-go v0.12.6 defined cliOverrides as a full command replacement. Keep that
			// public contract while the v0.13 base builder treats the same field as container args.
			mainContainer.Command = slices.Clone(buildCtx.MergedConfig.CliArgs)
			mainContainer.Args = nil
		} else {
			mainContainer.Args = []string{mainContainerScript(authentication)}
		}
	}

	labels := cloneStringMap(resources.StatefulSet.Labels)
	labels["prometheus.io/scrape"] = stringTrue
	resources.MetricsService = builder.NewMetricsServiceBuilder(
		buildCtx.ResourceName,
		buildCtx.ClusterNamespace,
		trinov1alpha1.MetricsPort,
		labels,
	).
		WithSelector(h.SelectorLabels(buildCtx)).
		WithTargetPortName(trinov1alpha1.MetricsPortName).
		Build()
	if resources.MetricsService.Annotations == nil {
		resources.MetricsService.Annotations = map[string]string{}
	}
	resources.MetricsService.Annotations["prometheus.io/scrape"] = stringTrue
	resources.MetricsService.Annotations["prometheus.io/port"] = strconv.Itoa(int(trinov1alpha1.MetricsPort))
	resources.MetricsService.Annotations["prometheus.io/scheme"] = "http"

	return resources, nil
}

func trinoProbe(portName string, scheme corev1.URIScheme) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/info",
				Port:   intstr.FromString(portName),
				Scheme: scheme,
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		FailureThreshold:    6,
		SuccessThreshold:    1,
	}
}

func roleConfigDefaults(cr *trinov1alpha1.TrinoCluster) *commonsv1alpha1.RoleGroupConfigSpec {
	capacity := resource.MustParse(defaultStorageCapacity)
	defaults := &commonsv1alpha1.RoleGroupConfigSpec{
		Resources: &commonsv1alpha1.ResourcesSpec{
			Storage: &commonsv1alpha1.StorageResource{Capacity: &capacity},
		},
	}
	if cr.Spec.ClusterConfig != nil && cr.Spec.ClusterConfig.VectorAggregatorConfigMapName != "" {
		enabled := true
		defaults.Logging = &commonsv1alpha1.LoggingSpec{EnableVectorAgent: &enabled}
	}
	return defaults
}

func tlsEnabled(cr *trinov1alpha1.TrinoCluster) bool {
	return cr.Spec.ClusterConfig != nil && cr.Spec.ClusterConfig.Tls != nil
}

func tlsSecretClasses(cr *trinov1alpha1.TrinoCluster) (string, string) {
	serverClass := trinov1alpha1.DefaultTlsSecretClass
	if tlsEnabled(cr) {
		if cr.Spec.ClusterConfig.Tls.ServerSecretClass != "" {
			serverClass = cr.Spec.ClusterConfig.Tls.ServerSecretClass
		}
	}

	// The legacy controller exposed and defaulted internalSecretClass but never used
	// it: serverSecretClass backed both TLS volumes for every persisted CR. Keep that
	// behavior during the framework migration. Activating the dormant field requires
	// a separate compatibility design because its existing value cannot prove intent.
	return serverClass, serverClass
}

func vectorEnabled(buildCtx *reconciler.RoleGroupBuildContext) bool {
	config := buildCtx.EffectiveConfig()
	return config != nil && config.Logging != nil && ptr.Deref(config.Logging.EnableVectorAgent, false)
}

func resolveAuthentication(
	ctx context.Context,
	k8sClient client.Client,
	cr *trinov1alpha1.TrinoCluster,
) (*authz.TrinoAuthentication, error) {
	if cr.Spec.ClusterConfig == nil || len(cr.Spec.ClusterConfig.Authentication) == 0 {
		return nil, nil
	}
	if !tlsEnabled(cr) {
		return nil, fmt.Errorf("trino authentication requires clusterConfig.tls")
	}
	return authz.NewAuthentication(ctx, k8sClient, cr.Namespace, cr.Spec.ClusterConfig.Authentication)
}

func mainContainerScript(authentication *authz.TrinoAuthentication) string {
	steps := []string{
		"set -e",
		"mkdir -p " + constant.KubedoopConfigDir,
		"cp -RL " + path.Join(constant.KubedoopConfigDirMount, "*") + " " + constant.KubedoopConfigDir,
		"mkdir -p " + path.Join(constant.KubedoopConfigDir, "catalog"),
		`for file in ` + path.Join(constant.KubedoopConfigDir, "catalog-*") + `; do
  if [ -f "$file" ]; then
    filename=$(basename "$file")
    mv "$file" "` + path.Join(constant.KubedoopConfigDir, "catalog") + `/$(echo "$filename" | sed 's/^catalog-//')"
  fi
done`,
		"rm -f " + path.Join(clientTLSMountPath, "truststore.p12"),
		"keytool -importkeystore" +
			" -srckeystore /etc/pki/java/cacerts" +
			" -srcstoretype JKS" +
			" -srcstorepass " + defaultTLSPassphrase +
			" -destkeystore " + path.Join(clientTLSMountPath, "truststore.p12") +
			" -deststoretype PKCS12" +
			" -deststorepass " + defaultTLSPassphrase +
			" -noprompt",
	}
	if authentication != nil {
		steps = append(steps, authentication.GetCommands()...)
	}
	steps = append(steps,
		"exec bin/launcher run --etc-dir "+constant.KubedoopConfigDir+" --data-dir "+constant.KubedoopDataDir,
	)
	return strings.Join(steps, "\n")
}

func findContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func podOverrideOwnsEntrypoint(buildCtx *reconciler.RoleGroupBuildContext, containerName string) bool {
	if buildCtx == nil || buildCtx.MergedConfig == nil || buildCtx.MergedConfig.PodOverrides == nil {
		return false
	}
	for _, container := range buildCtx.MergedConfig.PodOverrides.Spec.Containers {
		// operator-go treats an unnamed override container as the primary container for
		// backwards compatibility. Match that normalization here before deciding whether
		// the product may install its default entrypoint.
		if (container.Name == "" || container.Name == containerName) &&
			(container.Command != nil || container.Args != nil) {
			return true
		}
	}
	return false
}

type staticVolumeProvider struct {
	volumes []corev1.Volume
	mounts  []corev1.VolumeMount
}

func (p *staticVolumeProvider) Volumes() []corev1.Volume           { return p.volumes }
func (p *staticVolumeProvider) VolumeMounts() []corev1.VolumeMount { return p.mounts }
func (p *staticVolumeProvider) add(volume corev1.Volume, mount corev1.VolumeMount) {
	p.volumes = append(p.volumes, volume)
	p.mounts = append(p.mounts, mount)
}

func logVolume() corev1.Volume {
	return corev1.Volume{
		Name: logVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: ptr.To(resource.MustParse(defaultStorageCapacity))},
		},
	}
}

func clientTLSVolume() corev1.Volume {
	return corev1.Volume{
		Name: clientTLSVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: ptr.To(resource.MustParse("1Mi"))},
		},
	}
}

func secretTLSVolume(name, secretClass string, serviceNames ...string) corev1.Volume {
	scopeEntries := make([]string, 0, 2+len(serviceNames))
	scopeEntries = append(
		scopeEntries,
		string(security.PodScope),
		string(security.NodeScope),
	)
	for _, serviceName := range serviceNames {
		scopeEntries = append(scopeEntries, string(security.ServiceScope)+"="+serviceName)
	}

	provisioner := security.NewSecretProvisioner().Register(
		security.TLS(name, secretClass).
			WithScope(strings.Join(scopeEntries, security.CommonDelimiter)).
			WithPassword(defaultTLSPassphrase).
			WithStorageSize("1Mi"),
	)
	return provisioner.Volumes()[0]
}
