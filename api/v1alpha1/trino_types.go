/*
Copyright 2023 zncdata-labs.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TrinoSpec defines the desired state of Trino
type TrinoSpec struct {
	Image ImageSpec `json:"image,omitempty"`

	// +kubebuilder:validation=Optional
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	Resource *corev1.ResourceRequirements `json:"resource,omitempty"`

	// +kubebuilder:validation:Optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// +kubebuilder:validation:Optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// +kubebuilder:validation:Optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:validation:Optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// +kubebuilder:validation:Optional
	Ingress *IngressSpec `json:"ingress,omitempty"`
}

func (instance *Trino) GetNameWithSuffix(name string) string {
	return instance.GetName() + "-" + name
}

type ImageSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=trinodb/trino
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=latest
	Tag string `json:"tag,omitempty"`

	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +kubebuilder:default=IfNotPresent
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

type ServiceSpec struct {
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +kubebuilder:validation:enum=ClusterIP;NodePort;LoadBalancer;ExternalName
	// +kubebuilder:default=ClusterIP
	Type corev1.ServiceType `json:"type,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=18080
	Port int32 `json:"port,omitempty"`
}

type IngressSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=spark-history.example.com
	Host string `json:"host,omitempty"`
}

type PersistenceSpec struct {
	// +kubebuilder:validation:Optional
	StorageClass *string `json:"storageClass,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={ReadWriteOnce}
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// +kubebuilder:default="10Gi"
	Size string `json:"size,omitempty"`

	// +kubebuilder:validation:Optional
	ExistingClaim *string `json:"existingClaim,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=Filesystem
	VolumeMode *corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`

	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GetPvcName returns the name of the PVC for the HiveMetastore.
func (instance *Trino) GetPvcName() string {
	return instance.GetNameWithSuffix("pvc")
}

// TrinoStatus defines the observed state of Trino
type TrinoStatus struct {
	Nodes      []string                    `json:"nodes"`
	Conditions []corev1.ComponentCondition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Trino is the Schema for the trinoes API
type Trino struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoSpec   `json:"spec,omitempty"`
	Status TrinoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TrinoList contains a list of Trino
type TrinoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trino `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Trino{}, &TrinoList{})
}
