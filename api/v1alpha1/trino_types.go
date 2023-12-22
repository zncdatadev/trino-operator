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
	"github.com/zncdata-labs/operator-go/pkg/image"
	"github.com/zncdata-labs/operator-go/pkg/ingress"
	"github.com/zncdata-labs/operator-go/pkg/service"
	"github.com/zncdata-labs/operator-go/pkg/status"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TrinoSpec defines the desired state of Trino
type TrinoSpec struct {
	// +kubebuilder:validation:Required
	Image *image.ImageSpec `json:"image"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext"`

	// +kubebuilder:validation:Optional
	Tolerations *corev1.Toleration `json:"tolerations"`

	// +kubebuilder:validation:Optional
	Service *service.ServiceSpec `json:"service"`

	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels"`

	// +kubebuilder:validation:Optional
	Ingress *ingress.IngressSpec `json:"ingress"`

	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations"`

	// +kubebuilder:validation:Optional
	Server *ServerSpec `json:"server"`

	// +kubebuilder:validation:Required
	Coordinator *CoordinatorSpec `json:"coordinator"`

	// +kubebuilder:validation:Required
	Worker *WorkerSpec `json:"worker"`

	// +kubebuilder:validation:Optional
	Catalogs map[string]string `json:"catalogs"`
}

func (r *Trino) GetNameWithSuffix(suffix string) string {
	// return sparkHistory.GetName() + rand.String(5) + suffix
	return r.GetName() + "-" + suffix
}

type IngressSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Optional
	TLS *networkingv1.IngressTLS `json:"tls,omitempty"`
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="spark-history-server.example.com"
	Host string `json:"host,omitempty"`
}

type ServerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=2
	Worker          int32                `json:"worker"`
	Node            *NodeSpec            `json:"node,omitempty"`
	Config          *ConfigServerSpec    `json:"config"`
	ExchangeManager *ExchangeManagerSpec `json:"exchangeManager"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=INFO
	LogLevel string `json:"logLevel"`
}

type ExchangeManagerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="filesystem"
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/tmp/trino-local-file-system-exchange-manager"
	BaseDir string `json:"baseDir"`
}

type NodeSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="production"
	Environment string `json:"environment"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=/data/trino
	DataDir string `json:"dataDir"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=/usr/lib/trino/plugin
	PluginDir string `json:"pluginDir"`
}

type ConfigServerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/etc/trino"
	Path string `json:"path"`
	// +kubebuilder:validation:Optional
	Https *HttpsSpec `json:"https"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="4GB"
	QueryMaxMemory string `json:"queryMaxMemory"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	AuthenticationType string `json:"authenticationType"`
}

type HttpsSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=8443
	Port int `json:"port"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="4Gi"
	QueryMaxMemory string `json:"queryMaxMemory"`
}

type CoordinatorSpec struct {
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector"`

	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity"`

	// +kubebuilder:validation:Optional
	Tolerations *corev1.Toleration `json:"tolerations"`

	// +kubebuilder:validation:Required
	Resources *corev1.ResourceRequirements `json:"resources"`

	// +kubebuilder:validation:Optional
	Jvm *JvmCoordinatorSpec `json:"jvm,omitempty"`

	// +kubebuilder:validation:Optional
	Config *ConfigCoordinatorSpec `json:"config"`
}

type JvmCoordinatorSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="8Gi"
	MaxHeapSize string `json:"maxHeapSize"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="UseG1GC"
	GcMethodType string `json:"gcMethodType"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="32M"
	G1HeapRegionSize string `json:"gcHeapRegionSize"`
}

type ConfigCoordinatorSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	MemoryHeapHeadroomPerNode string `json:"memoryHeapHeadroomPerNode"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="1GB"
	QueryMaxMemoryPerNode string `json:"queryMaxMemoryPerNode"`
}

type WorkerSpec struct {
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector"`

	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity"`

	// +kubebuilder:validation:Optional
	Tolerations *corev1.Toleration `json:"tolerations"`

	// +kubebuilder:validation:Required
	Resources *corev1.ResourceRequirements `json:"resources"`

	// +kubebuilder:validation:Optional
	Jvm *JvmWorkerSpec `json:"jvm,omitempty"`

	// +kubebuilder:validation:Optional
	Config *ConfigWrokerSpec `json:"config"`
}

type JvmWorkerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="8Gi"
	MaxHeapSize string `json:"maxHeapSize"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="UseG1GC"
	GcMethodType string `json:"gcMethodType"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="32M"
	G1HeapRegionSize string `json:"gcHeapRegionSize"`
}

type ConfigWrokerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	MemoryHeapHeadroomPerNode string `json:"memoryHeapHeadroomPerNode"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="1GB"
	QueryMaxMemoryPerNode string `json:"queryMaxMemoryPerNode"`
}

// SetStatusCondition updates the status condition using the provided arguments.
// If the condition already exists, it updates the condition; otherwise, it appends the condition.
// If the condition status has changed, it updates the condition's LastTransitionTime.
func (r *Trino) SetStatusCondition(condition metav1.Condition) {
	r.Status.SetStatusCondition(condition)
}

// InitStatusConditions initializes the status conditions to the provided conditions.
func (r *Trino) InitStatusConditions() {
	r.Status.InitStatus(r)
	r.Status.InitStatusConditions()
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Trino is the Schema for the trinoes API
type Trino struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoSpec            `json:"spec,omitempty"`
	Status status.ZncdataStatus `json:"status,omitempty"`
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
