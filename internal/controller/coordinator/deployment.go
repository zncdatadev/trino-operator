package coordinator

import (
	"context"
	"github.com/zncdatadev/trino-operator/internal/util"
	"k8s.io/apimachinery/pkg/api/resource"
	"maps"
	"time"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentReconciler struct {
	common.WorkloadStyleReconciler[*trinov1alpha1.TrinoCluster, *trinov1alpha1.RoleGroupSpec]
}

// NewDeployment new a DeploymentReconcile
func NewDeployment(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	mergedLabels map[string]string,
	mergedCfg *trinov1alpha1.RoleGroupSpec,
	replicates int32,
) *DeploymentReconciler {
	return &DeploymentReconciler{
		WorkloadStyleReconciler: *common.NewDeploymentStyleReconciler(
			scheme,
			instance,
			client,
			groupName,
			mergedLabels,
			mergedCfg,
			replicates,
		),
	}
}

// GetConditions implement the ConditionGetter interface
func (d *DeploymentReconciler) GetConditions() *[]metav1.Condition {
	return &d.Instance.Status.Conditions
}

// Build implements the ResourceBuilder interface
func (d *DeploymentReconciler) Build(ctx context.Context) (client.Object, error) {
	podTemplate := d.getPodTemplate()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      createCoordinatorDeploymentName(d.Instance.Name, d.GroupName),
			Namespace: d.Instance.Namespace,
			Labels:    d.MergedLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: d.getReplicas(ctx),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.MergedLabels,
			},
			Template: podTemplate,
		},
	}
	coordinatorWithVector(d.MergedCfg.Config.Logging, dep, createCoordinatorConfigmapName(d.Instance.Name, d.GroupName))
	return dep, nil
}

func (d *DeploymentReconciler) SetAffinity(resource client.Object) {
	dep := resource.(*appsv1.Deployment)
	if affinity := d.MergedCfg.Config.Affinity; affinity != nil {
		dep.Spec.Template.Spec.Affinity = affinity
	} else {
		dep.Spec.Template.Spec.Affinity = common.AffinityDefault(common.Coordinator, d.Instance.GetName())
	}
}

// CommandOverride implement the WorkloadOverride interface
func (d *DeploymentReconciler) CommandOverride(resource client.Object) {
	dep := resource.(*appsv1.Deployment)
	containers := dep.Spec.Template.Spec.Containers
	if cmdOverride := d.MergedCfg.CommandArgsOverrides; cmdOverride != nil {
		for i := range containers {
			containers[i].Command = cmdOverride
		}
	}
}

// EnvOverride implement the WorkloadOverride interface
func (d *DeploymentReconciler) EnvOverride(resource client.Object) {
	dep := resource.(*appsv1.Deployment)
	containers := dep.Spec.Template.Spec.Containers
	if envOverride := d.MergedCfg.EnvOverrides; envOverride != nil {
		for i := range containers {
			envVars := containers[i].Env
			common.OverrideEnvVars(&envVars, d.MergedCfg.EnvOverrides)
		}
	}
}

// LogOverride implement the WorkloadOverride interface
func (d *DeploymentReconciler) LogOverride(resource client.Object) {
	if d.isLoggersOverrideEnabled() {
		d.logVolumesOverride(resource)
		d.logVolumeMountsOverride(resource)
	}
}

func (d *DeploymentReconciler) getPodTemplate() corev1.PodTemplateSpec {
	copyedPodTemplate := d.MergedCfg.PodOverride.DeepCopy()
	podTemplate := corev1.PodTemplateSpec{}

	if copyedPodTemplate != nil {
		podTemplate = *copyedPodTemplate
	}

	if podTemplate.ObjectMeta.Labels == nil {
		podTemplate.ObjectMeta.Labels = make(map[string]string)
	}

	maps.Copy(podTemplate.ObjectMeta.Labels, d.MergedLabels)

	podTemplate.Spec.Containers = d.getContainers()

	podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, d.createVolumes()...)

	seconds := d.getTerminationGracePeriodSeconds()
	if d.MergedCfg.Config.GracefulShutdownTimeout != nil {
		podTemplate.Spec.TerminationGracePeriodSeconds = seconds
	}
	return podTemplate
}

func (d *DeploymentReconciler) getContainers() []corev1.Container {
	resourceSpec := d.MergedCfg.Config.Resources
	imageSpec := d.getImageSpec()
	image := util.ImageRepository(imageSpec.Repository, imageSpec.Tag)
	coordinator := NewCoordinatorContainerBuilder(image, imageSpec.PullPolicy, resourceSpec)
	return []corev1.Container{
		coordinator.Build(coordinator),
	}
}

func (d *DeploymentReconciler) getTerminationGracePeriodSeconds() *int64 {
	if d.MergedCfg.Config.GracefulShutdownTimeout != nil {
		if tiime, err := time.ParseDuration(*d.MergedCfg.Config.GracefulShutdownTimeout); err == nil {
			seconds := int64(tiime.Seconds())
			return &seconds
		}
	}
	return nil
}

// is loggers override enabled
func (d *DeploymentReconciler) isLoggersOverrideEnabled() bool {
	return d.MergedCfg.Config.Logging != nil && d.MergedCfg.Config.Logging.Trino != nil
}

func (d *DeploymentReconciler) logVolumesOverride(resource client.Object) {
	dep := resource.(*appsv1.Deployment)
	volumes := dep.Spec.Template.Spec.Volumes
	if len(volumes) == 0 {
		volumes = make([]corev1.Volume, 1)
	}
	volumes = append(volumes, corev1.Volume{
		Name: d.logConfigVolumeName(),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: common.CreateRoleGroupLoggingConfigMapName(d.Instance.Name, string(common.Coordinator),
						d.GroupName),
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "log.properties",
						Path: "log.properties",
					},
				},
			},
		},
	})
	dep.Spec.Template.Spec.Volumes = volumes
}

func (d *DeploymentReconciler) logVolumeMountsOverride(resource client.Object) {
	dep := resource.(*appsv1.Deployment)
	containers := dep.Spec.Template.Spec.Containers
	for i := range containers {
		containers[i].VolumeMounts = append(containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      d.logConfigVolumeName(),
			MountPath: "/etc/trino/log.properties",
			SubPath:   "log.properties",
		})
	}
}

// create volumes
func (d *DeploymentReconciler) createVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: common.ConfigVolumeName(),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: createCoordinatorConfigmapName(d.Instance.GetName(), d.GroupName),
					},
				},
			},
		},
		{
			Name: common.CatalogVolumeName(),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: common.CreateCatalogConfigmapName(d.Instance.GetName()),
					},
				},
			},
		},
		{
			Name: common.SchemaVolumeName(),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: common.CreateSchemaConfigmapName(d.Instance.GetName()),
					},
				},
			},
		},
		{
			Name: common.LogVolumeName(),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: func() *resource.Quantity {
						r := resource.MustParse("33Mi")
						return &r
					}(),
				},
			},
		},
	}
}

func (d *DeploymentReconciler) getImageSpec() *trinov1alpha1.ImageSpec {
	return d.Instance.Spec.Image
}

// get replicas
func (d *DeploymentReconciler) getReplicas(ctx context.Context) *int32 {
	if d.shouldStop(ctx) {
		logger.Info("Stop the cluster, set replicas to 0")
		reps := int32(0)
		return &reps
	}
	return &d.MergedCfg.Replicas
}

func (d *DeploymentReconciler) shouldStop(ctx context.Context) bool {

	clusterOperation := common.NewClusterOperation(
		&common.TrinoInstance{Instance: d.Instance},
		common.ResourceClient{
			Ctx:       ctx,
			Client:    d.Client,
			Namespace: d.Instance.Namespace,
		},
	)

	return clusterOperation.ClusterStop()
}

// create log config volume name
func (d *DeploymentReconciler) logConfigVolumeName() string {
	return "log-config"
}
