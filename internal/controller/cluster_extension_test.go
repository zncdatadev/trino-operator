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
	"bytes"
	"context"
	"encoding/base64"
	"maps"
	"testing"

	"github.com/zncdatadev/operator-go/pkg/listener"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

func TestClusterResourcesExtension_CoordinatorServiceContract(t *testing.T) {
	tests := []struct {
		name          string
		listenerClass listener.ListenerClass
		tls           bool
		wantType      corev1.ServiceType
		wantPortName  string
		wantPort      int32
	}{
		{
			name:         "internal HTTP",
			wantType:     corev1.ServiceTypeClusterIP,
			wantPortName: trinov1alpha1.HttpPortName,
			wantPort:     trinov1alpha1.HttpPort,
		},
		{
			name:          "unstable external TLS",
			listenerClass: listener.ListenerClassExternalUnstable,
			tls:           true,
			wantType:      corev1.ServiceTypeNodePort,
			wantPortName:  trinov1alpha1.HttpsPortName,
			wantPort:      trinov1alpha1.HttpsPort,
		},
		{
			name:          "stable external HTTP",
			listenerClass: listener.ListenerClassExternalStable,
			wantType:      corev1.ServiceTypeLoadBalancer,
			wantPortName:  trinov1alpha1.HttpPortName,
			wantPort:      trinov1alpha1.HttpPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newControllerTestCluster()
			cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
				ListenerClass: tt.listenerClass,
			}
			if tt.tls {
				cr.Spec.ClusterConfig.Tls = &trinov1alpha1.TlsSpec{}
			}
			scheme := controllerTestScheme(t)
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()
			extension := NewClusterResourcesExtension(scheme)

			if err := extension.PreReconcile(context.Background(), k8sClient, cr); err != nil {
				t.Fatalf("PreReconcile() error = %v", err)
			}

			service := &corev1.Service{}
			key := client.ObjectKey{
				Namespace: cr.Namespace,
				Name:      cr.Name + "-" + trinov1alpha1.TrinoCoordinatorRoleName,
			}
			if err := k8sClient.Get(context.Background(), key, service); err != nil {
				t.Fatalf("get coordinator Service: %v", err)
			}
			wantSelector := map[string]string{
				reconciler.ClusterLabelKey(LabelDomain): cr.Name,
				reconciler.RoleLabelKey(LabelDomain):    trinov1alpha1.TrinoCoordinatorRoleName,
			}
			if !maps.Equal(service.Spec.Selector, wantSelector) {
				t.Errorf("Service selector = %#v, want %#v", service.Spec.Selector, wantSelector)
			}
			if service.Spec.Type != tt.wantType {
				t.Errorf("Service type = %q, want %q", service.Spec.Type, tt.wantType)
			}
			if len(service.Spec.Ports) != 1 {
				t.Fatalf("Service ports = %#v, want one application port", service.Spec.Ports)
			}
			port := service.Spec.Ports[0]
			if port.Name != tt.wantPortName ||
				port.Port != tt.wantPort ||
				port.TargetPort.Type != intstr.String ||
				port.TargetPort.StrVal != tt.wantPortName ||
				port.Protocol != corev1.ProtocolTCP {
				t.Errorf("Service port = %#v, want TCP %s:%d targeting the named pod port",
					port, tt.wantPortName, tt.wantPort)
			}
			assertControllerOwner(t, service.OwnerReferences, cr)
		})
	}
}

func TestClusterResourcesExtension_GeneratedSecretRemainsStable(t *testing.T) {
	cr := newControllerTestCluster()
	cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
		Tls: &trinov1alpha1.TlsSpec{},
	}
	scheme := controllerTestScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()
	extension := NewClusterResourcesExtension(scheme)
	ctx := context.Background()

	if err := extension.PreReconcile(ctx, k8sClient, cr); err != nil {
		t.Fatalf("first PreReconcile() error = %v", err)
	}
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: cr.Namespace, Name: internalSharedSecretName(cr.Name)}
	if err := k8sClient.Get(ctx, key, secret); err != nil {
		t.Fatalf("get generated Secret after first reconcile: %v", err)
	}
	firstValue := bytes.Clone(secret.Data[internalSharedSecretEnvName])
	if len(firstValue) == 0 {
		t.Fatal("generated Secret contains an empty internal shared secret")
	}
	decoded, err := base64.StdEncoding.DecodeString(string(firstValue))
	if err != nil {
		t.Fatalf("generated Secret value is not the legacy base64 shape: %v", err)
	}
	if len(decoded) != 512 {
		t.Errorf("decoded generated Secret length = %d, want 512 bytes", len(decoded))
	}
	assertControllerOwner(t, secret.OwnerReferences, cr)

	if err := extension.PreReconcile(ctx, k8sClient, cr); err != nil {
		t.Fatalf("second PreReconcile() error = %v", err)
	}
	if err := k8sClient.Get(ctx, key, secret); err != nil {
		t.Fatalf("get generated Secret after second reconcile: %v", err)
	}
	if !bytes.Equal(secret.Data[internalSharedSecretEnvName], firstValue) {
		t.Error("internal shared secret rotated on reconcile")
	}
}

func TestClusterResourcesExtension_PreservesAllocatedNodePort(t *testing.T) {
	cr := newControllerTestCluster()
	cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
		ListenerClass: listener.ListenerClassExternalUnstable,
	}
	serviceName := cr.Name + "-" + trinov1alpha1.TrinoCoordinatorRoleName
	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: cr.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Name:       trinov1alpha1.HttpPortName,
				Port:       trinov1alpha1.HttpPort,
				TargetPort: intstr.FromString(trinov1alpha1.HttpPortName),
				NodePort:   32080,
			}},
		},
	}
	scheme := controllerTestScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, existing).Build()
	extension := NewClusterResourcesExtension(scheme)

	if err := extension.PreReconcile(context.Background(), k8sClient, cr); err != nil {
		t.Fatalf("PreReconcile() error = %v", err)
	}
	service := &corev1.Service{}
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(existing), service); err != nil {
		t.Fatalf("get reconciled coordinator Service: %v", err)
	}
	if len(service.Spec.Ports) != 1 {
		t.Fatalf("Service ports = %#v, want one application port", service.Spec.Ports)
	}
	if got := service.Spec.Ports[0].NodePort; got != 32080 {
		t.Errorf("allocated NodePort = %d, want preserved value 32080", got)
	}
}

func TestClusterResourcesExtension_PreservesForeignServiceAnnotations(t *testing.T) {
	const sharedAnnotation = "example.com/shared"

	cr := newControllerTestCluster()
	cr.Annotations = map[string]string{
		"example.com/from-cluster": "desired",
		sharedAnnotation:           "cluster-wins",
	}
	serviceName := cr.Name + "-" + trinov1alpha1.TrinoCoordinatorRoleName
	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: cr.Namespace,
			Annotations: map[string]string{
				"example.com/foreign": "preserved",
				sharedAnnotation:      "live-value",
			},
		},
	}
	scheme := controllerTestScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, existing).Build()
	extension := NewClusterResourcesExtension(scheme)

	if err := extension.PreReconcile(context.Background(), k8sClient, cr); err != nil {
		t.Fatalf("PreReconcile() error = %v", err)
	}
	service := &corev1.Service{}
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(existing), service); err != nil {
		t.Fatalf("get reconciled coordinator Service: %v", err)
	}
	want := map[string]string{
		"example.com/foreign":      "preserved",
		"example.com/from-cluster": "desired",
		sharedAnnotation:           "cluster-wins",
	}
	if !maps.Equal(service.Annotations, want) {
		t.Errorf("Service annotations = %#v, want %#v", service.Annotations, want)
	}
}

func assertControllerOwner(t *testing.T, owners []metav1.OwnerReference, cr *trinov1alpha1.TrinoCluster) {
	t.Helper()
	for _, owner := range owners {
		if owner.Controller != nil && *owner.Controller &&
			owner.UID == cr.UID &&
			owner.Name == cr.Name &&
			owner.Kind == "TrinoCluster" {
			return
		}
	}
	t.Errorf("owner references %#v do not contain the TrinoCluster controller owner", owners)
}
