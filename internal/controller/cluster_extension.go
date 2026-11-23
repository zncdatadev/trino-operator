/*
Copyright 2026 zncdatadev.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"

	opcommon "github.com/zncdatadev/operator-go/pkg/common"
	"github.com/zncdatadev/operator-go/pkg/constant"
	"github.com/zncdatadev/operator-go/pkg/listener"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const (
	clusterResourcesExtensionName = "trino-cluster-resources"
	frameworkManagedBy            = "operator-go"
)

var _ opcommon.ClusterExtension[*trinov1alpha1.TrinoCluster] = &ClusterResourcesExtension{}

// ClusterResourcesExtension maintains the resources shared by every coordinator role group:
// the stable client Service and, for TLS clusters, the internal-communication secret.
type ClusterResourcesExtension struct {
	scheme *runtime.Scheme
}

func NewClusterResourcesExtension(scheme *runtime.Scheme) *ClusterResourcesExtension {
	return &ClusterResourcesExtension{scheme: scheme}
}

func (e *ClusterResourcesExtension) Name() string { return clusterResourcesExtensionName }

func (e *ClusterResourcesExtension) PreReconcile(
	ctx context.Context,
	k8sClient client.Client,
	cr *trinov1alpha1.TrinoCluster,
) error {
	if err := e.ensureCoordinatorService(ctx, k8sClient, cr); err != nil {
		return err
	}
	if !tlsEnabled(cr) {
		return nil
	}

	_, err := reconciler.EnsureGeneratedSecret(
		ctx,
		k8sClient,
		e.scheme,
		cr,
		internalSharedSecretName(cr.Name),
		map[string]func() (string, error){internalSharedSecretEnvName: generateInternalSharedSecret},
		reconciler.WithGeneratedSecretProductName(trinov1alpha1.DefaultProductName),
		reconciler.WithGeneratedSecretExtraLabels(map[string]string{
			reconciler.ClusterLabelKey(LabelDomain): cr.Name,
		}),
	)
	if err != nil {
		return fmt.Errorf("ensure internal communication secret: %w", err)
	}
	return nil
}

func (e *ClusterResourcesExtension) PostReconcile(
	context.Context,
	client.Client,
	*trinov1alpha1.TrinoCluster,
) error {
	return nil
}

func (e *ClusterResourcesExtension) OnReconcileError(
	context.Context,
	client.Client,
	*trinov1alpha1.TrinoCluster,
	error,
) error {
	return nil
}

func (e *ClusterResourcesExtension) ensureCoordinatorService(
	ctx context.Context,
	k8sClient client.Client,
	cr *trinov1alpha1.TrinoCluster,
) error {
	portName := trinov1alpha1.HttpPortName
	port := trinov1alpha1.HttpPort
	if tlsEnabled(cr) {
		portName = trinov1alpha1.HttpsPortName
		port = trinov1alpha1.HttpsPort
	}

	listenerClass := trinov1alpha1.DefaultListenerClass
	if cr.Spec.ClusterConfig != nil && cr.Spec.ClusterConfig.ListenerClass != "" {
		listenerClass = cr.Spec.ClusterConfig.ListenerClass
	}

	labels := maps.Clone(cr.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[constant.LabelKubernetesInstance] = cr.Name
	labels[constant.LabelKubernetesName] = trinov1alpha1.DefaultProductName
	labels[constant.LabelKubernetesManagedBy] = frameworkManagedBy
	labels[constant.LabelKubernetesComponent] = trinov1alpha1.TrinoCoordinatorRoleName
	labels[reconciler.ClusterLabelKey(LabelDomain)] = cr.Name
	labels[reconciler.RoleLabelKey(LabelDomain)] = trinov1alpha1.TrinoCoordinatorRoleName

	selector := map[string]string{
		reconciler.ClusterLabelKey(LabelDomain): cr.Name,
		reconciler.RoleLabelKey(LabelDomain):    trinov1alpha1.TrinoCoordinatorRoleName,
	}
	service := &corev1.Service{}
	service.Name = cr.Name + "-" + trinov1alpha1.TrinoCoordinatorRoleName
	service.Namespace = cr.Namespace

	_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, service, func() error {
		allocatedNodePorts := make(map[string]int32, len(service.Spec.Ports))
		for _, existingPort := range service.Spec.Ports {
			if existingPort.NodePort != 0 {
				allocatedNodePorts[existingPort.Name] = existingPort.NodePort
			}
		}

		service.Labels = labels
		annotations := maps.Clone(service.Annotations)
		if len(cr.Annotations) > 0 {
			if annotations == nil {
				annotations = make(map[string]string, len(cr.Annotations))
			}
			maps.Copy(annotations, cr.Annotations)
		}
		service.Annotations = annotations
		service.Spec.Selector = selector
		desiredType := listener.ServiceTypeFor(listenerClass)
		service.Spec.Type = desiredType
		service.Spec.Ports = []corev1.ServicePort{{
			Name:       portName,
			Port:       port,
			TargetPort: intstr.FromString(portName),
			Protocol:   corev1.ProtocolTCP,
		}}
		if desiredType == corev1.ServiceTypeNodePort || desiredType == corev1.ServiceTypeLoadBalancer {
			service.Spec.Ports[0].NodePort = allocatedNodePorts[portName]
		}
		if desiredType != corev1.ServiceTypeLoadBalancer {
			service.Spec.AllocateLoadBalancerNodePorts = nil
			service.Spec.LoadBalancerClass = nil
			service.Spec.HealthCheckNodePort = 0
		}
		if desiredType == corev1.ServiceTypeClusterIP {
			service.Spec.ExternalTrafficPolicy = ""
		}
		return controllerutil.SetControllerReference(cr, service, e.scheme)
	})
	if err != nil {
		return fmt.Errorf("ensure coordinator service %q: %w", service.Name, err)
	}
	return nil
}

func internalSharedSecretName(clusterName string) string {
	return clusterName + "-internal-shared-secret"
}

func generateInternalSharedSecret() (string, error) {
	randomData := make([]byte, 512)
	if _, err := rand.Read(randomData); err != nil {
		return "", err
	}
	// Preserve the legacy value shape: a base64 string stored as the Secret value. Kubernetes
	// itself base64-encodes Secret data once more when serializing the object.
	return base64.StdEncoding.EncodeToString(randomData), nil
}
