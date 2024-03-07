package coordinator

import (
	"context"
	trinov1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	"github.com/zncdata-labs/trino-operator/internal/common"
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
func (d *DeploymentReconciler) Build(_ context.Context) (client.Object, error) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      createCoordinatorDeploymentName(d.Instance.Name, d.GroupName),
			Namespace: d.Instance.Namespace,
			Labels:    d.MergedLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: d.getReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.MergedLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: d.MergedLabels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: d.getSecurityContext(),
					Containers: []corev1.Container{
						d.createContainer(),
					},
					Volumes: d.createVolumes(),
				},
			},
		},
	}
	return dep, nil
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

// is loggers override enabled
func (d *DeploymentReconciler) isLoggersOverrideEnabled() bool {
	return d.MergedCfg.Config.Logging != nil
}

func (d *DeploymentReconciler) logVolumesOverride(resource client.Object) {
	dep := resource.(*appsv1.Deployment)
	volumes := dep.Spec.Template.Spec.Volumes
	if len(volumes) == 0 {
		volumes = make([]corev1.Volume, 1)
	}
	volumes = append(volumes, corev1.Volume{
		Name: d.logVolumeName(),
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
			Name:      d.logVolumeName(),
			MountPath: "/etc/trino/log.properties",
			SubPath:   "log.properties",
		})
	}
}

// create container
func (d *DeploymentReconciler) createContainer() corev1.Container {
	image := d.getImageSpec()
	return corev1.Container{
		Name:            d.Instance.GetNameWithSuffix("coordinator"),
		Image:           image.Repository + ":" + image.Tag,
		ImagePullPolicy: image.PullPolicy,
		Resources:       d.getResources(),
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 18080,
				Name:          "http",
				Protocol:      "TCP",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      d.configVolumeName(),
				MountPath: "/etc/trino",
			},
			{
				Name:      d.catalogVolumeName(),
				MountPath: "/etc/trino/catalog",
			},
			{
				Name:      d.schemaVolumeName(),
				MountPath: "/etc/trino/schemas",
			},
		},
	}
}

// create volumes
func (d *DeploymentReconciler) createVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: d.configVolumeName(),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: createCoordinatorConfigmapName(d.Instance.GetName(), d.GroupName),
					},
				},
			},
		},
		{
			Name: d.catalogVolumeName(),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: common.CreateCatalogConfigmapName(d.Instance.GetName(), d.GroupName),
					},
				},
			},
		},
		{
			Name: d.schemaVolumeName(),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: common.CreateSchemaConfigmapName(d.Instance.GetName(), d.GroupName),
					},
				},
			},
		},
	}
}

func (d *DeploymentReconciler) getImageSpec() *trinov1alpha1.ImageSpec {
	return d.Instance.Spec.ClusterConfig.Image
}

// get security context
func (d *DeploymentReconciler) getSecurityContext() *corev1.PodSecurityContext {
	return d.MergedCfg.Config.SecurityContext
}

// get replicas
func (d *DeploymentReconciler) getReplicas() *int32 {
	return &d.MergedCfg.Replicas
}

// get resources
func (d *DeploymentReconciler) getResources() corev1.ResourceRequirements {
	resourcesSpec := d.MergedCfg.Config.Resources
	return *common.ConvertToResourceRequirements(resourcesSpec)
}

func (d *DeploymentReconciler) configVolumeName() string {
	return "config-volume"
}

func (d *DeploymentReconciler) catalogVolumeName() string {
	return "catalog-volume"
}

func (d *DeploymentReconciler) schemaVolumeName() string {
	return "schema-volume"
}

// create log4j2 volume name
func (d *DeploymentReconciler) logVolumeName() string {
	return "log-volume"
}
