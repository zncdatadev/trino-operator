package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zncdata-labs/operator-go/pkg/errors"
	"github.com/zncdata-labs/operator-go/pkg/status"
	"github.com/zncdata-labs/operator-go/pkg/utils"

	stackv1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *TrinoReconciler) makeIngress(instance *stackv1alpha1.TrinoCluster) ([]*v1.Ingress, error) {
	var ing []*v1.Ingress

	if instance.Spec.Coordinator.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Coordinator.RoleGroups {
			i, err := r.makeIngressForRoleGroup(instance, roleGroupName, roleGroup, r.Scheme)
			if err != nil {
				return nil, err
			}
			ing = append(ing, i)
		}
	}
	return ing, nil
}

func (r *TrinoReconciler) makeIngressForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupCoordinatorSpec, schema *runtime.Scheme) (*v1.Ingress, error) {
	labels := instance.GetLabels()

	additionalLabels := make(map[string]string)

	if roleGroup.Config != nil && roleGroup.Config.MatchLabels != nil {
		for k, v := range roleGroup.Config.MatchLabels {
			additionalLabels[k] = v
		}
	}

	mergedLabels := make(map[string]string)
	for key, value := range labels {
		mergedLabels[key] = value
	}
	for key, value := range additionalLabels {
		mergedLabels[key] = value
	}

	pt := v1.PathTypeImplementationSpecific

	var host string
	var port int32

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.Ingress != nil {
		if !roleGroup.Config.Ingress.Enabled {
			return nil, nil
		}
		host = roleGroup.Config.Ingress.Host
		if roleGroup.Config.Service != nil {
			port = roleGroup.Config.Service.Port
		} else if instance.Spec.Service != nil {
			port = instance.Spec.Service.Port
		}
	} else {
		if instance.Spec.Ingress != nil && !instance.Spec.Ingress.Enabled {
			return nil, nil
		}
		host = instance.Spec.Ingress.Host
		if instance.Spec.Service != nil {
			port = instance.Spec.Service.Port
		}
	}

	ing := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix(roleGroupName),
			Namespace: instance.Namespace,
			Labels:    mergedLabels,
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pt,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: instance.GetNameWithSuffix(roleGroupName),
											Port: v1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := ctrl.SetControllerReference(instance, ing, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for ingress")
		return nil, errors.Wrap(err, "Failed to set controller reference for ingress")
	}
	return ing, nil
}

func (r *TrinoReconciler) reconcileIngress(ctx context.Context, instance *stackv1alpha1.TrinoCluster) error {
	obj, err := r.makeIngress(instance)
	if err != nil {
		return err
	}
	if r != nil && r.Client != nil {
		for _, ingress := range obj {
			if ingress != nil {
				if err := CreateOrUpdate(ctx, r.Client, ingress); err != nil {
					r.Log.Error(err, "Failed to create or update ingress")
					return err
				}
			}
		}
	}

	if instance.Spec.Ingress.Enabled {
		url := fmt.Sprintf("http://%s", instance.Spec.Ingress.Host)
		if instance.Status.URLs == nil {
			instance.Status.URLs = []status.URL{
				{
					Name: "webui",
					URL:  url,
				},
			}
			if err := utils.UpdateStatus(ctx, r.Client, instance); err != nil {
				return err
			}

		} else if instance.Spec.Ingress.Host != instance.Status.URLs[0].Name {
			instance.Status.URLs[0].URL = url
			if err := utils.UpdateStatus(ctx, r.Client, instance); err != nil {
				return err
			}

		}
	}

	return nil
}

func (r *TrinoReconciler) makeServices(instance *stackv1alpha1.TrinoCluster) ([]*corev1.Service, error) {
	var services []*corev1.Service

	if instance.Spec.Coordinator.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Coordinator.RoleGroups {
			svc, err := r.makeServiceForRoleGroup(instance, roleGroupName, roleGroup, r.Scheme)
			if err != nil {
				return nil, err
			}
			services = append(services, svc)
		}
	}

	return services, nil
}

func (r *TrinoReconciler) makeServiceForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupCoordinatorSpec, schema *runtime.Scheme) (*corev1.Service, error) {
	labels := instance.GetLabels()

	additionalLabels := make(map[string]string)

	if roleGroup.Config != nil && roleGroup.Config.MatchLabels != nil {
		for k, v := range roleGroup.Config.MatchLabels {
			additionalLabels[k] = v
		}
	}

	mergedLabels := make(map[string]string)
	for key, value := range labels {
		mergedLabels[key] = value
	}
	for key, value := range additionalLabels {
		mergedLabels[key] = value
	}

	var port int32
	var serviceType corev1.ServiceType
	var annotations map[string]string

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.Service != nil {
		port = roleGroup.Config.Service.Port
		serviceType = roleGroup.Config.Service.Type
		annotations = roleGroup.Config.Service.Annotations
	} else {
		port = instance.Spec.Service.Port
		serviceType = instance.Spec.Service.Type
		annotations = instance.Spec.Service.Annotations
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.GetNameWithSuffix(roleGroupName),
			Namespace:   instance.Namespace,
			Labels:      mergedLabels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     port,
					Name:     "http",
					Protocol: "TCP",
				},
			},
			Selector: mergedLabels,
			Type:     serviceType,
		},
	}
	err := ctrl.SetControllerReference(instance, svc, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for service")
		return nil, errors.Wrap(err, "Failed to set controller reference for service")
	}
	return svc, nil
}

func (r *TrinoReconciler) reconcileService(ctx context.Context, instance *stackv1alpha1.TrinoCluster) error {
	services, err := r.makeServices(instance)
	if err != nil {
		return err
	}

	for _, svc := range services {
		if svc == nil {
			continue
		}

		if err := CreateOrUpdate(ctx, r.Client, svc); err != nil {
			r.Log.Error(err, "Failed to create or update service", "service", svc.Name)
			return err
		}
	}

	return nil
}

func (r *TrinoReconciler) makeCoordinatorDeployments(instance *stackv1alpha1.TrinoCluster) []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	if instance.Spec.Coordinator.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Coordinator.RoleGroups {
			dep := r.makeCoordinatorDeploymentForRoleGroup(instance, roleGroupName, roleGroup, r.Scheme)
			if dep != nil {
				deployments = append(deployments, dep)
			}
		}
	}

	return deployments
}

func (r *TrinoReconciler) makeCoordinatorDeploymentForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupCoordinatorSpec, schema *runtime.Scheme) *appsv1.Deployment {
	labels := instance.GetLabels()

	additionalLabels := make(map[string]string)

	if roleGroup != nil && roleGroup.Config.MatchLabels != nil {
		for k, v := range roleGroup.Config.MatchLabels {
			additionalLabels[k] = v
		}
	}

	mergedLabels := make(map[string]string)
	for key, value := range labels {
		mergedLabels[key] = value
	}
	for key, value := range additionalLabels {
		mergedLabels[key] = value
	}

	var image stackv1alpha1.ImageSpec
	var securityContext *corev1.PodSecurityContext

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.Image != nil {
		image = *roleGroup.Config.Image
	} else {
		image = *instance.Spec.Image
	}

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.SecurityContext != nil {
		securityContext = roleGroup.Config.SecurityContext
	} else {
		securityContext = instance.Spec.SecurityContext
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("coordinator-" + roleGroupName),
			Namespace: instance.Namespace,
			Labels:    mergedLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &roleGroup.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: mergedLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergedLabels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: securityContext,
					Containers: []corev1.Container{
						{
							Name:            instance.GetNameWithSuffix("coordinator"),
							Image:           image.Repository + ":" + image.Tag,
							ImagePullPolicy: image.PullPolicy,
							Resources:       *roleGroup.Config.Resources,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 18080,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/trino",
								},
								{
									Name:      "catalog-volume",
									MountPath: "/etc/trino/catalog",
								},
								{
									Name:      "schemas-volume",
									MountPath: "/etc/trino/schemas",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.GetNameWithSuffix("coordinator" + "-" + roleGroupName),
									},
								},
							},
						},
						{
							Name: "catalog-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.GetNameWithSuffix("catalog"),
									},
								},
							},
						},
						{
							Name: "schemas-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.GetNameWithSuffix("schemas"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	CoordinatorScheduler(dep, roleGroup)

	err := ctrl.SetControllerReference(instance, dep, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for Coordinator deployment")
		return nil
	}
	return dep
}

func (r *TrinoReconciler) makeWorkerDeployments(instance *stackv1alpha1.TrinoCluster) []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	if instance.Spec.Worker.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Worker.RoleGroups {
			dep := r.makeWorkerDeploymentForRoleGroup(instance, roleGroupName, roleGroup, r.Scheme)
			if dep != nil {
				deployments = append(deployments, dep)
			}
		}
	}

	return deployments
}

func (r *TrinoReconciler) makeWorkerDeploymentForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupsWorkerSpec, schema *runtime.Scheme) *appsv1.Deployment {
	labels := instance.GetLabels()

	additionalLabels := make(map[string]string)

	if roleGroup != nil && roleGroup.Config.MatchLabels != nil {
		for k, v := range roleGroup.Config.MatchLabels {
			additionalLabels[k] = v
		}
	}

	mergedLabels := make(map[string]string)
	for key, value := range labels {
		mergedLabels[key] = value
	}
	for key, value := range additionalLabels {
		mergedLabels[key] = value
	}

	var image stackv1alpha1.ImageSpec
	var securityContext *corev1.PodSecurityContext

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.Image != nil {
		image = *roleGroup.Config.Image
	} else {
		image = *instance.Spec.Image
	}

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.SecurityContext != nil {
		securityContext = roleGroup.Config.SecurityContext
	} else {
		securityContext = instance.Spec.SecurityContext
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("worker-" + roleGroupName),
			Namespace: instance.Namespace,
			Labels:    mergedLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &roleGroup.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: mergedLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergedLabels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: securityContext,
					Containers: []corev1.Container{
						{
							Name:            instance.GetNameWithSuffix("worker"),
							Image:           image.Repository + ":" + image.Tag,
							ImagePullPolicy: image.PullPolicy,
							Resources:       *roleGroup.Config.Resources,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 18080,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/trino",
								},
								{
									Name:      "catalog-volume",
									MountPath: "/etc/trino/catalog",
								},
								{
									Name:      "schemas-volume",
									MountPath: "/etc/trino/schemas",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.GetNameWithSuffix("worker" + "-" + roleGroupName),
									},
								},
							},
						},
						{
							Name: "catalog-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.GetNameWithSuffix("catalog"),
									},
								},
							},
						},
						{
							Name: "schemas-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.GetNameWithSuffix("schemas"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	WorkerScheduler(dep, roleGroup)

	err := ctrl.SetControllerReference(instance, dep, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for worker deployment")
		return nil
	}
	return dep
}

func (r *TrinoReconciler) reconcileDeployment(ctx context.Context, instance *stackv1alpha1.TrinoCluster) error {
	// 处理协调器部署
	coordinatorDeployments := r.makeCoordinatorDeployments(instance)
	for _, dep := range coordinatorDeployments {
		if dep == nil {
			continue
		}

		if err := CreateOrUpdate(ctx, r.Client, dep); err != nil {
			r.Log.Error(err, "Failed to create or update coordinator Deployment", "deployment", dep.Name)
			return err
		}
	}

	// 处理工作器部署
	workerDeployments := r.makeWorkerDeployments(instance)
	for _, dep := range workerDeployments {
		if dep == nil {
			continue
		}

		if instance.Spec.ClusterConfig.ClusterMode {
			if err := CreateOrUpdate(ctx, r.Client, dep); err != nil {
				r.Log.Error(err, "Failed to create or update worker Deployment", "deployment", dep.Name)
				return err
			}
		}
	}

	return nil
}

func (r *TrinoReconciler) makeCoordinatorConfigMaps(instance *stackv1alpha1.TrinoCluster) []*corev1.ConfigMap {
	var configMaps []*corev1.ConfigMap

	if instance.Spec.Coordinator.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Coordinator.RoleGroups {
			cm := r.makeCoordinatorConfigMapForRoleGroup(instance, roleGroupName, roleGroup, r.Scheme)
			if cm != nil {
				configMaps = append(configMaps, cm)
			}
		}
	}

	return configMaps
}

func (r *TrinoReconciler) makeCoordinatorConfigMapForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupCoordinatorSpec, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()

	var jvmProperties *stackv1alpha1.JvmPropertiesRoleConfigSpec
	var configProperties *stackv1alpha1.ConfigPropertiesSpec
	var svc *stackv1alpha1.ServiceSpec

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.JvmProperties != nil {
		jvmProperties = roleGroup.Config.JvmProperties
	} else {
		jvmProperties = instance.Spec.Coordinator.RoleConfig.JvmProperties
	}

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.ConfigProperties != nil {
		configProperties = roleGroup.Config.ConfigProperties
	} else {
		configProperties = instance.Spec.Coordinator.RoleConfig.ConfigProperties
	}

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.Service != nil {
		svc = roleGroup.Config.Service
	} else {
		svc = instance.Spec.Service
	}

	nodeProps := "node.environment=" + instance.Spec.ClusterConfig.NodeProperties.Environment + "\n" +
		"node.data-dir=" + instance.Spec.ClusterConfig.NodeProperties.DataDir + "\n" +
		"plugin.dir=" + instance.Spec.ClusterConfig.NodeProperties.PluginDir + "\n"

	jvmConfigData := "-server\n" +
		"-Xmx" + jvmProperties.MaxHeapSize + "\n" +
		"-XX:+" + jvmProperties.GcMethodType + "\n" +
		"-XX:G1HeapRegionSize=" + jvmProperties.G1HeapRegionSize + "\n" +
		"-XX:+UseGCOverheadLimit\n" +
		"-XX:+ExplicitGCInvokesConcurrent\n" +
		"-XX:+HeapDumpOnOutOfMemoryError\n" +
		"-XX:+ExitOnOutOfMemoryError\n" +
		"-Djdk.attach.allowAttachSelf=true\n" +
		"-XX:-UseBiasedLocking\n" +
		"-XX:ReservedCodeCacheSize=512M\n" +
		"-XX:PerMethodRecompilationCutoff=10000\n" +
		"-XX:PerBytecodeRecompilationCutoff=10000\n" +
		"-Djdk.nio.maxCachedBufferSize=2000000\n" +
		"-XX:+UnlockDiagnosticVMOptions\n" +
		"-XX:+UseAESCTRIntrinsics\n"

	configProps := "coordinator=true\n" +
		"http-server.http.port=" + strconv.Itoa(int(svc.Port)) + "\n" +
		"query.max-memory=" + instance.Spec.ClusterConfig.ConfigProperties.QueryMaxMemory + "\n" +
		"query.max-memory-per-node=" + configProperties.QueryMaxMemoryPerNode + "\n" +
		"discovery.uri=http://localhost:" + strconv.Itoa(int(svc.Port)) + "\n"

	if instance.Spec.ClusterConfig.ClusterMode {
		configProps += "node-scheduler.include-coordinator=false" + "\n"
	} else {
		configProps += "node-scheduler.include-coordinator=true" + "\n"
	}

	if configProperties.MemoryHeapHeadroomPerNode != "" {
		configProps += "memory.heap-headroom-per-node=" + configProperties.MemoryHeapHeadroomPerNode + "\n"
	}

	if instance.Spec.ClusterConfig.ConfigProperties.AuthenticationType != "" {
		configProps += "http-server.authentication.type=" + instance.Spec.ClusterConfig.ConfigProperties.AuthenticationType + "\n"
	}

	exchangeManagerProps := "exchange-manager.name=" + instance.Spec.ClusterConfig.ExchangeManager.Name + "\n"

	if instance.Spec.ClusterConfig.ExchangeManager.Name == "filesystem" {
		exchangeManagerProps += "exchange.base-directories=" + instance.Spec.ClusterConfig.ExchangeManager.BaseDir
	}

	logProps := "io.trino=" + instance.Spec.ClusterConfig.LogLevel + "\n"

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("coordinator" + "-" + roleGroupName),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"node.properties":             nodeProps,
			"jvm.config":                  jvmConfigData,
			"config.properties":           configProps,
			"exchange-manager.properties": exchangeManagerProps,
			"log.properties":              logProps,
		},
	}
	err := ctrl.SetControllerReference(instance, &cm, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for configmap")
		return nil
	}
	return &cm
}

func (r *TrinoReconciler) makeWorkerConfigMaps(instance *stackv1alpha1.TrinoCluster) []*corev1.ConfigMap {
	var configMaps []*corev1.ConfigMap

	if instance.Spec.Worker.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Worker.RoleGroups {
			cm := r.makeWorkerConfigMapForRoleGroup(instance, roleGroupName, roleGroup, r.Scheme)
			if cm != nil {
				configMaps = append(configMaps, cm)
			}
		}
	}

	return configMaps
}

func (r *TrinoReconciler) makeWorkerConfigMapForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupsWorkerSpec, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()

	var jvmProperties *stackv1alpha1.JvmPropertiesRoleConfigSpec
	var configProperties *stackv1alpha1.ConfigPropertiesSpec
	var svc *stackv1alpha1.ServiceSpec

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.JvmProperties != nil {
		jvmProperties = roleGroup.Config.JvmProperties
	} else {
		jvmProperties = instance.Spec.Coordinator.RoleConfig.JvmProperties
	}

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.ConfigProperties != nil {
		configProperties = roleGroup.Config.ConfigProperties
	} else {
		configProperties = instance.Spec.Coordinator.RoleConfig.ConfigProperties
	}

	if roleGroup != nil && roleGroup.Config != nil && roleGroup.Config.Service != nil {
		svc = roleGroup.Config.Service
	} else {
		svc = instance.Spec.Service
	}

	nodeProps := "node.environment=" + instance.Spec.ClusterConfig.NodeProperties.Environment + "\n" +
		"node.data-dir=" + instance.Spec.ClusterConfig.NodeProperties.DataDir + "\n" +
		"plugin.dir=" + instance.Spec.ClusterConfig.NodeProperties.PluginDir + "\n"

	jvmConfigData := "-server\n" +
		"-Xmx" + jvmProperties.MaxHeapSize + "\n" +
		"-XX:+" + jvmProperties.GcMethodType + "\n" +
		"-XX:G1HeapRegionSize=" + jvmProperties.G1HeapRegionSize + "\n" +
		"-XX:+UseGCOverheadLimit\n" +
		"-XX:+ExplicitGCInvokesConcurrent\n" +
		"-XX:+HeapDumpOnOutOfMemoryError\n" +
		"-XX:+ExitOnOutOfMemoryError\n" +
		"-Djdk.attach.allowAttachSelf=true\n" +
		"-XX:-UseBiasedLocking\n" +
		"-XX:ReservedCodeCacheSize=512M\n" +
		"-XX:PerMethodRecompilationCutoff=10000\n" +
		"-XX:PerBytecodeRecompilationCutoff=10000\n" +
		"-Djdk.nio.maxCachedBufferSize=2000000\n" +
		"-XX:+UnlockDiagnosticVMOptions\n" +
		"-XX:+UseAESCTRIntrinsics\n"

	configProps := "coordinator=false\n" +
		"http-server.http.port=" + strconv.Itoa(int(svc.Port)) + "\n" +
		"query.max-memory=" + instance.Spec.ClusterConfig.ConfigProperties.QueryMaxMemory + "\n" +
		"query.max-memory-per-node=" + configProperties.QueryMaxMemoryPerNode + "\n" +
		"discovery.uri=http://" + instance.GetNameWithSuffix(roleGroupName) + ":" + strconv.Itoa(int(svc.Port)) + "\n"

	if configProperties.MemoryHeapHeadroomPerNode != "" {
		configProps += "memory.heap-headroom-per-node=" + configProperties.MemoryHeapHeadroomPerNode + "\n"
	}

	if instance.Spec.ClusterConfig.ConfigProperties.AuthenticationType != "" {
		configProps += "http-server.authentication.type=" + instance.Spec.ClusterConfig.ConfigProperties.AuthenticationType + "\n"
	}

	exchangeManagerProps := "exchange-manager.name=" + instance.Spec.ClusterConfig.ExchangeManager.Name + "\n"

	if instance.Spec.ClusterConfig.ExchangeManager.Name == "filesystem" {
		exchangeManagerProps += "exchange.base-directories=" + instance.Spec.ClusterConfig.ExchangeManager.BaseDir
	}

	logProps := "io.trino=" + instance.Spec.ClusterConfig.LogLevel + "\n"

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("worker" + "-" + roleGroupName),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"node.properties":             nodeProps,
			"jvm.config":                  jvmConfigData,
			"config.properties":           configProps,
			"exchange-manager.properties": exchangeManagerProps,
			"log.properties":              logProps,
		},
	}
	err := ctrl.SetControllerReference(instance, &cm, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set worker reference for configmap")
		return nil
	}
	return &cm
}

// 缩进属性
func indentProperties(properties string, spaces int) string {
	indented := ""
	for _, line := range splitLines(properties) {
		indented += fmt.Sprintf("%s%s\n", strings.Repeat("", spaces), line)
	}
	return indented
}

// 按行拆分字符串
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func (r *TrinoReconciler) makeCatalogConfigMap(instance *stackv1alpha1.TrinoCluster, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()

	//hiveName, hivePort := r.GetHiveMetastoreList()

	tpchProps := "connector.name=tpch\n" +
		"tpch.splits-per-node=4\n"

	tpcdsProps := "connector.name=tpcds\n" +
		"tpcds.splits-per-node=4\n"

	additionalCatalogs := make(map[string]string)
	for catalogName, catalogProperties := range instance.Spec.ClusterConfig.Catalogs {
		key := fmt.Sprintf("%s.properties", catalogName)
		additionalCatalogs[key] = fmt.Sprintf("%s\n", indentProperties(catalogProperties, 4))
	}

	data := map[string]string{
		"tpch.properties":  tpchProps,
		"tpcds.properties": tpcdsProps,
	}

	for key, value := range additionalCatalogs {
		data[key] = value
	}

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("catalog"),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
	}
	err := ctrl.SetControllerReference(instance, &cm, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set catalog reference for configmap")
		return nil
	}
	return &cm
}

func (r *TrinoReconciler) makeSchemasConfigMap(instance *stackv1alpha1.TrinoCluster, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("schemas"),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	}
	err := ctrl.SetControllerReference(instance, &cm, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set schemas reference for configmap")
		return nil
	}
	return &cm
}

func (r *TrinoReconciler) reconcileConfigMap(ctx context.Context, instance *stackv1alpha1.TrinoCluster) error {

	CoordinatorConfigMap := r.makeCoordinatorConfigMaps(instance)
	for _, cm := range CoordinatorConfigMap {
		if cm == nil {
			continue
		}

		if err := CreateOrUpdate(ctx, r.Client, cm); err != nil {
			r.Log.Error(err, "Failed to create or update coordinator configmap", "configmap", cm.Name)
			return err
		}
	}

	WorkerConfigMap := r.makeWorkerConfigMaps(instance)
	for _, cm := range WorkerConfigMap {
		if cm == nil {
			continue
		}

		if err := CreateOrUpdate(ctx, r.Client, cm); err != nil {
			r.Log.Error(err, "Failed to create or update worker configmap", "configmap", cm.Name)
			return err
		}
	}

	CatalogConfigMap := r.makeCatalogConfigMap(instance, r.Scheme)
	if CatalogConfigMap == nil {
		return nil
	}

	SchemasConfigMap := r.makeSchemasConfigMap(instance, r.Scheme)
	if SchemasConfigMap == nil {
		return nil
	}

	catalogConfigMap := r.makeCatalogConfigMap(instance, r.Scheme)
	if catalogConfigMap != nil {
		if err := CreateOrUpdate(ctx, r.Client, catalogConfigMap); err != nil {
			r.Log.Error(err, "Failed to create or update catalog configmap")
			return err
		}
	}

	schemasConfigMap := r.makeSchemasConfigMap(instance, r.Scheme)
	if schemasConfigMap != nil {
		if err := CreateOrUpdate(ctx, r.Client, schemasConfigMap); err != nil {
			r.Log.Error(err, "Failed to create or update schemas configmap")
			return err
		}
	}
	return nil
}
