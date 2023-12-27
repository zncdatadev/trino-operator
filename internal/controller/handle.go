package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	stackv1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *TrinoReconciler) makeIngress(instance *stackv1alpha1.TrinoCluster, schema *runtime.Scheme) *v1.Ingress {
	labels := instance.GetLabels()

	pt := v1.PathTypeImplementationSpecific

	ing := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: instance.Spec.Ingress.Host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pt,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: instance.GetName(),
											Port: v1.ServiceBackendPort{
												Number: instance.Spec.Service.Port,
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
		return nil
	}
	return ing
}

func (r *TrinoReconciler) reconcileIngress(ctx context.Context, instance *stackv1alpha1.TrinoCluster) error {
	obj := r.makeIngress(instance, r.Scheme)
	if obj == nil {
		return nil
	}

	if err := CreateOrUpdate(ctx, r.Client, obj); err != nil {
		r.Log.Error(err, "Failed to create or update ingress")
		return err
	}

	if instance.Spec.Ingress.Enabled {
		url := fmt.Sprintf("http://%s", instance.Spec.Ingress.Host)
		if instance.Status.URLs == nil {
			instance.Status.URLs = []stackv1alpha1.StatusURL{
				{
					Name: "webui",
					URL:  url,
				},
			}
			if err := r.UpdateStatus(ctx, instance); err != nil {
				return err
			}
		} else if instance.Spec.Ingress.Host != instance.Status.URLs[0].Name {
			instance.Status.URLs[0].URL = url
			if err := r.UpdateStatus(ctx, instance); err != nil {
				return err
			}
		}
	}
	return nil
}

// make service
func (r *TrinoReconciler) makeService(instance *stackv1alpha1.TrinoCluster, schema *runtime.Scheme) *corev1.Service {
	labels := instance.GetLabels()
	labels["component"] = "coordinator"

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Spec.Service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     instance.Spec.Service.Port,
					Name:     "http",
					Protocol: "TCP",
				},
			},
			Selector: labels,
			Type:     instance.Spec.Service.Type,
		},
	}
	err := ctrl.SetControllerReference(instance, svc, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for service")
		return nil
	}
	return svc
}

func (r *TrinoReconciler) reconcileService(ctx context.Context, instance *stackv1alpha1.TrinoCluster) error {
	obj := r.makeService(instance, r.Scheme)
	if obj == nil {
		return nil
	}

	if err := CreateOrUpdate(ctx, r.Client, obj); err != nil {
		r.Log.Error(err, "Failed to create or update service")
		return err
	}
	return nil
}

func (r *TrinoReconciler) makeCoordinatorDeployments(instance *stackv1alpha1.TrinoCluster) []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	if instance.Spec.Coordinator.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Coordinator.RoleGroups {
			if roleGroup != nil {
				if instance.Spec.Coordinator.Selectors != nil {
					for _, selectors := range instance.Spec.Coordinator.Selectors {
						if selectors != nil {
							dep := r.makeCoordinatorDeploymentForRoleGroup(instance, roleGroupName, roleGroup, selectors, r.Scheme)
							if dep != nil {
								deployments = append(deployments, dep)
							}
						}
					}
				}
			}
		}
	}

	return deployments
}

func (r *TrinoReconciler) makeCoordinatorDeploymentForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupCoordinatorSpec, selectors *stackv1alpha1.SelectorSpec, schema *runtime.Scheme) *appsv1.Deployment {
	labels := instance.GetLabels()

	additionalLabels := make(map[string]string)

	if instance.Spec.Coordinator.Selectors != nil {
		for _, selectorSpec := range instance.Spec.Coordinator.Selectors {
			if selectorSpec != nil && selectorSpec.Selector.MatchLabels != nil {
				for k, v := range selectorSpec.Selector.MatchLabels {
					additionalLabels[k] = v
				}
			}
		}
	}

	mergedLabels := make(map[string]string)
	for key, value := range labels {
		mergedLabels[key] = value
	}
	for key, value := range additionalLabels {
		mergedLabels[key] = value
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
					SecurityContext: instance.Spec.SecurityContext,
					Containers: []corev1.Container{
						{
							Name:            instance.GetNameWithSuffix("coordinator"),
							Image:           instance.Spec.Image.Repository + ":" + instance.Spec.Image.Tag,
							ImagePullPolicy: instance.Spec.Image.PullPolicy,
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
										Name: instance.GetNameWithSuffix("coordinator"),
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
	if &selectors.NodeSelector != nil {
		dep.Spec.Template.Spec.NodeSelector = selectors.NodeSelector
	}

	CoordinatorScheduler(instance, dep, roleGroup)

	err := ctrl.SetControllerReference(instance, dep, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for Coordinator deployment")
		return nil
	}
	return dep
}

func (r *TrinoReconciler) updateStatusConditionWithDeployment(ctx context.Context, instance *stackv1alpha1.TrinoCluster, status metav1.ConditionStatus, message string) error {
	instance.SetStatusCondition(metav1.Condition{
		Type:               stackv1alpha1.ConditionTypeProgressing,
		Status:             status,
		Reason:             stackv1alpha1.ConditionReasonReconcileDeployment,
		Message:            message,
		ObservedGeneration: instance.GetGeneration(),
		LastTransitionTime: metav1.Now(),
	})

	if err := r.UpdateStatus(ctx, instance); err != nil {
		return err
	}
	return nil
}

func (r *TrinoReconciler) makeWorkerDeployments(instance *stackv1alpha1.TrinoCluster) []*appsv1.Deployment {
	var deployments []*appsv1.Deployment

	if instance.Spec.Worker.RoleGroups != nil {
		for roleGroupName, roleGroup := range instance.Spec.Worker.RoleGroups {
			if roleGroup != nil {
				if instance.Spec.Worker.Selectors != nil {
					for _, selectors := range instance.Spec.Worker.Selectors {
						if selectors != nil {
							dep := r.makeWorkerDeploymentForRoleGroup(instance, roleGroupName, roleGroup, selectors, r.Scheme)
							if dep != nil {
								deployments = append(deployments, dep)
							}
						}
					}
				}
			}
		}
	}

	return deployments
}

func (r *TrinoReconciler) makeWorkerDeploymentForRoleGroup(instance *stackv1alpha1.TrinoCluster, roleGroupName string, roleGroup *stackv1alpha1.RoleGroupsWorkerSpec, selectors *stackv1alpha1.SelectorSpec, schema *runtime.Scheme) *appsv1.Deployment {
	labels := instance.GetLabels()

	additionalLabels := make(map[string]string)

	if instance.Spec.Worker.Selectors != nil {
		for _, selectorSpec := range instance.Spec.Worker.Selectors {
			if selectorSpec != nil && selectorSpec.Selector.MatchLabels != nil {
				for k, v := range selectorSpec.Selector.MatchLabels {
					additionalLabels[k] = v
				}
			}
		}
	}

	mergedLabels := make(map[string]string)
	for key, value := range labels {
		mergedLabels[key] = value
	}
	for key, value := range additionalLabels {
		mergedLabels[key] = value
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
					SecurityContext: instance.Spec.SecurityContext,
					Containers: []corev1.Container{
						{
							Name:            instance.GetNameWithSuffix("worker"),
							Image:           instance.Spec.Image.Repository + ":" + instance.Spec.Image.Tag,
							ImagePullPolicy: instance.Spec.Image.PullPolicy,
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
										Name: instance.GetNameWithSuffix("coordinator"),
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

	if selectors.NodeSelector != nil {
		dep.Spec.Template.Spec.NodeSelector = selectors.NodeSelector
	}

	WorkerScheduler(instance, dep, roleGroup)

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

func (r *TrinoReconciler) makeCoordinatorConfigMap(instance *stackv1alpha1.TrinoCluster, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()

	nodeProps := "node.environment=" + instance.Spec.ClusterConfig.NodeProperties.Environment + "\n" +
		"node.data-dir=" + instance.Spec.ClusterConfig.NodeProperties.DataDir + "\n" +
		"plugin.dir=" + instance.Spec.ClusterConfig.NodeProperties.PluginDir + "\n"

	jvmConfigData := "-server\n" +
		"-Xmx" + instance.Spec.Coordinator.RoleConfig.JvmProperties.MaxHeapSize + "\n" +
		"-XX:+" + instance.Spec.Coordinator.RoleConfig.JvmProperties.GcMethodType + "\n" +
		"-XX:G1HeapRegionSize=" + instance.Spec.Coordinator.RoleConfig.JvmProperties.G1HeapRegionSize + "\n" +
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
		"http-server.http.port=" + strconv.Itoa(int(instance.Spec.Service.Port)) + "\n" +
		"query.max-memory=" + instance.Spec.ClusterConfig.ConfigProperties.QueryMaxMemory + "\n" +
		"query.max-memory-per-node=" + instance.Spec.Coordinator.RoleConfig.ConfigProperties.QueryMaxMemoryPerNode + "\n" +
		"discovery.uri=http://localhost:" + strconv.Itoa(int(instance.Spec.Service.Port)) + "\n"

	if instance.Spec.ClusterConfig.ClusterMode {
		configProps += "node-scheduler.include-coordinator=false" + "\n"
	} else {
		configProps += "node-scheduler.include-coordinator=true" + "\n"
	}

	if instance.Spec.Coordinator.RoleConfig.ConfigProperties.MemoryHeapHeadroomPerNode != "" {
		configProps += "memory.heap-headroom-per-node=" + instance.Spec.Coordinator.RoleConfig.ConfigProperties.MemoryHeapHeadroomPerNode + "\n"
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
			Name:      instance.GetNameWithSuffix("coordinator"),
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

func (r *TrinoReconciler) makeWorkerConfigMap(instance *stackv1alpha1.TrinoCluster, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()

	nodeProps := "node.environment=" + instance.Spec.ClusterConfig.NodeProperties.Environment + "\n" +
		"node.data-dir=" + instance.Spec.ClusterConfig.NodeProperties.DataDir + "\n" +
		"plugin.dir=" + instance.Spec.ClusterConfig.NodeProperties.PluginDir + "\n"

	jvmConfigData := "-server\n" +
		"-Xmx" + instance.Spec.Worker.RoleConfig.JvmProperties.MaxHeapSize + "\n" +
		"-XX:+" + instance.Spec.Worker.RoleConfig.JvmProperties.GcMethodType + "\n" +
		"-XX:G1HeapRegionSize=" + instance.Spec.Worker.RoleConfig.JvmProperties.G1HeapRegionSize + "\n" +
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
		"http-server.http.port=" + strconv.Itoa(int(instance.Spec.Service.Port)) + "\n" +
		"query.max-memory=" + instance.Spec.ClusterConfig.ConfigProperties.QueryMaxMemory + "\n" +
		"query.max-memory-per-node=" + instance.Spec.Worker.RoleConfig.ConfigProperties.QueryMaxMemoryPerNode + "\n" +
		"discovery.uri=http://" + instance.Name + ":" + strconv.Itoa(int(instance.Spec.Service.Port)) + "\n"

	if instance.Spec.Worker.RoleConfig.ConfigProperties.MemoryHeapHeadroomPerNode != "" {
		configProps += "memory.heap-headroom-per-node=" + instance.Spec.Worker.RoleConfig.ConfigProperties.MemoryHeapHeadroomPerNode + "\n"
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
			Name:      instance.GetNameWithSuffix("worker"),
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

	CoordinatorConfigMap := r.makeCoordinatorConfigMap(instance, r.Scheme)
	if CoordinatorConfigMap == nil {
		return nil
	}

	WorkerConfigMap := r.makeWorkerConfigMap(instance, r.Scheme)
	if WorkerConfigMap == nil {
		return nil
	}

	CatalogConfigMap := r.makeCatalogConfigMap(instance, r.Scheme)
	if CatalogConfigMap == nil {
		return nil
	}

	SchemasConfigMap := r.makeSchemasConfigMap(instance, r.Scheme)
	if SchemasConfigMap == nil {
		return nil
	}

	coordinatorConfigMap := r.makeCoordinatorConfigMap(instance, r.Scheme)
	if coordinatorConfigMap != nil {
		if err := CreateOrUpdate(ctx, r.Client, coordinatorConfigMap); err != nil {
			r.Log.Error(err, "Failed to create or update coordinator configmap")
			return err
		}
	}

	workerConfigMap := r.makeWorkerConfigMap(instance, r.Scheme)
	if workerConfigMap != nil {
		if err := CreateOrUpdate(ctx, r.Client, workerConfigMap); err != nil {
			r.Log.Error(err, "Failed to create or update worker configmap")
			return err
		}
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
