package controller

import (
	"context"
	"encoding/json"
	"fmt"
	stackv1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
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
									Name:      instance.GetNameWithSuffix("config"),
									MountPath: "/etc/trino",
								},
								{
									Name:      instance.GetNameWithSuffix("catalog"),
									MountPath: "/etc/trino/catalog",
								},
								{
									Name:      instance.GetNameWithSuffix("schemas"),
									MountPath: "/etc/trino/schemas",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: instance.GetNameWithSuffix("config"),
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "trino-coordinator",
									},
								},
							},
						},
						{
							Name: instance.GetNameWithSuffix("catalog"),
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "trino-catalog",
									},
								},
							},
						},
						{
							Name: instance.GetNameWithSuffix("schemas"),
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "trino-schemas-coordinator",
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

func (r *TrinoReconciler) makeConfigMap(instance *stackv1alpha1.Trino, schema *runtime.Scheme) *corev1.ConfigMap {
	labels := instance.GetLabels()
	nodeProps := instance.Spec.Server.Node
	jvmConfigData := "-server\n" +
		"-Xmx" + instance.Spec.Coordinator.Jvm.MaxHeapSize + "\n" +
		"-XX:+Use" + instance.Spec.Coordinator.Jvm.GcMethod + "\n" +
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
		"-XX:+UseAESCTRIntrinsics"

	nodePropsJSON, setErr := json.Marshal(nodeProps)
	if setErr != nil {
		// Handle the error
	}

	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetNameWithSuffix("coordinator"),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"node.properties": string(nodePropsJSON),
			"jvm.config":      jvmConfigData,
		},
	}
	err := ctrl.SetControllerReference(instance, &cm, schema)
	if err != nil {
		r.Log.Error(err, "Failed to set controller reference for configmap")
		return nil
	}
	return &cm
}

func (r *TrinoReconciler) reconcileConfigMap(ctx context.Context, instance *stackv1alpha1.Trino) error {
	obj := r.makeConfigMap(instance, r.Scheme)
	if obj == nil {
		return nil
	}

	if err := CreateOrUpdate(ctx, r.Client, obj); err != nil {
		r.Log.Error(err, "Failed to create or update service")
		return err
	}
	return nil
}
