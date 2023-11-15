package controller

import (
	"context"
	"fmt"
	stackv1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

func (r *TrinoReconciler) makeIngress(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *v1.Ingress {
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

func (r *TrinoReconciler) reconcileIngress(ctx context.Context, instance *stackv1alpha1.Trino) error {
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
func (r *TrinoReconciler) makeService(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *corev1.Service {
	labels := instance.GetLabels()
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

func (r *TrinoReconciler) reconcileService(ctx context.Context, instance *stackv1alpha1.Trino) error {
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

func (r *TrinoReconciler) makeDeployment(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *appsv1.Deployment {
	labels := instance.GetLabels()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("coordinator"),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: instance.Spec.SecurityContext,
					Containers: []corev1.Container{
						{
							Name:            instance.GetNameWithSuffix("coordinator"),
							Image:           instance.Spec.Image.Repository + ":" + instance.Spec.Image.Tag,
							ImagePullPolicy: instance.Spec.Image.PullPolicy,
							Resources:       *instance.Spec.Resources,
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
	err := ctrl.SetControllerReference(instance, dep, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for deployment")
		return nil
	}
	return dep
}

func (r *TrinoReconciler) updateStatusConditionWithDeployment(ctx context.Context, instance *stackv1alpha1.Trino, status metav1.ConditionStatus, message string) error {
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

func (r *TrinoReconciler) reconcileDeployment(ctx context.Context, instance *stackv1alpha1.Trino) error {
	obj := r.makeDeployment(instance, r.Scheme)
	if obj == nil {
		return nil
	}
	if err := CreateOrUpdate(ctx, r.Client, obj); err != nil {
		logger.Error(err, "Failed to create or update deployment")
		return err
	}
	return nil
}

func (r *TrinoReconciler) makeCoordinatorConfigMap(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()

	nodeProps := "node.environment=" + instance.Spec.Server.Node.Environment + "\n" +
		"node.data-dir=" + instance.Spec.Server.Node.DataDir + "\n" +
		"plugin.dir=" + instance.Spec.Server.Node.PluginDir + "\n"

	jvmConfigData := "-server\n" +
		"-Xmx" + instance.Spec.Coordinator.Jvm.MaxHeapSize + "\n" +
		"-XX:+" + instance.Spec.Coordinator.Jvm.GcMethodType + "\n" +
		"-XX:G1HeapRegionSize=" + instance.Spec.Coordinator.Jvm.G1HeapRegionSize + "\n" +
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
		"query.max-memory=" + instance.Spec.Server.Config.QueryMaxMemory + "\n" +
		"query.max-memory-per-node=" + instance.Spec.Coordinator.Config.QueryMaxMemoryPerNode + "\n" +
		"discovery.uri=http://localhost:" + strconv.Itoa(int(instance.Spec.Service.Port)) + "\n"

	if instance.Spec.Server.Worker > 0 {
		configProps += "node-scheduler.include-coordinator=false" + "\n"
	} else {
		configProps += "node-scheduler.include-coordinator=true" + "\n"
	}

	if instance.Spec.Coordinator.Config.MemoryHeapHeadroomPerNode != "" {
		configProps += "memory.heap-headroom-per-node=" + instance.Spec.Coordinator.Config.MemoryHeapHeadroomPerNode + "\n"
	}

	if instance.Spec.Server.Config.AuthenticationType != "" {
		configProps += "http-server.authentication.type=" + instance.Spec.Server.Config.AuthenticationType + "\n"
	}

	exchangeManagerProps := "exchange-manager.name=" + instance.Spec.Server.ExchangeManager.Name + "\n"

	if instance.Spec.Server.ExchangeManager.Name == "filesystem" {
		exchangeManagerProps += "exchange.base-directories=" + instance.Spec.Server.ExchangeManager.BaseDir
	}

	logProps := "io.trino=INFO" + "\n"

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

func (r *TrinoReconciler) makeCatalogConfigMap(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()
	tpchProps := "connector.name=tpch\n" +
		"tpch.splits-per-node=4\n"

	tpcdsProps := "connector.name=tpcds\n" +
		"tpcds.splits-per-node=4\n"

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("catalog"),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"tpch.properties":  tpchProps,
			"tpcds.properties": tpcdsProps,
		},
	}
	err := ctrl.SetControllerReference(instance, &cm, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for configmap")
		return nil
	}
	return &cm
}

func (r *TrinoReconciler) makeSchemasConfigMap(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *corev1.ConfigMap {
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
		r.Log.Error(err, "Failed to set controller reference for configmap")
		return nil
	}
	return &cm
}

func (r *TrinoReconciler) reconcileConfigMap(ctx context.Context, instance *stackv1alpha1.Trino) error {
	CoordinatorConfigMap := r.makeCoordinatorConfigMap(instance, r.Scheme)
	if CoordinatorConfigMap == nil {
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

	// 创建第二个 ConfigMap
	catalogConfigMap := r.makeCatalogConfigMap(instance, r.Scheme)
	if catalogConfigMap != nil {
		if err := CreateOrUpdate(ctx, r.Client, catalogConfigMap); err != nil {
			r.Log.Error(err, "Failed to create or update catalog configmap")
			return err
		}
	}

	// 创建第三个 ConfigMap
	schemasConfigMap := r.makeSchemasConfigMap(instance, r.Scheme)
	if schemasConfigMap != nil {
		if err := CreateOrUpdate(ctx, r.Client, schemasConfigMap); err != nil {
			r.Log.Error(err, "Failed to create or update schemas configmap")
			return err
		}
	}
	return nil
}
