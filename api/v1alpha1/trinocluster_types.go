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
	"github.com/zncdata-labs/operator-go/pkg/status"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TrinoClusterSpec defines the desired state of TrinoCluster
type TrinoClusterSpec struct {
	// +kubebuilder:validation:Required
	Image *ImageSpec `json:"image"`

	// +kubebuilder:validation:Optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// +kubebuilder:validation:Optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +kubebuilder:validation:Required
	Coordinator *CoordinatorSpec `json:"coordinator"`

	// +kubebuilder:validation:Required
	Worker *WorkerSpec `json:"worker"`

	// +kubebuilder:validation:Optional
	ClusterConfig *ClusterConfigSpec `json:"clusterConfig,omitempty"`
}

type ClusterConfigSpec struct {
	// +kubebuilder:validation:Optional
	Catalogs map[string]string `json:"catalogs,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default:=true
	ClusterMode bool `json:"clusterMode"`

	// +kubebuilder:validation:Optional
	NodeProperties *NodePropertiesSpec `json:"nodeProperties,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigProperties *ConfigPropertiesSpec `json:"configProperties,omitempty"`

	// +kubebuilder:validation:Optional
	ExchangeManager *ExchangeManagerSpec `json:"exchangeManager,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=INFO
	LogLevel string `json:"logLevel,omitempty"`
}

func (r *TrinoCluster) GetNameWithSuffix(suffix string) string {
	// return sparkHistory.GetName() + rand.String(5) + suffix
	return r.GetName() + "-" + suffix
}

type ImageSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=trinodb/trino
	Repository string `json:"repository,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="423"
	Tag string `json:"tag,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=IfNotPresent
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

type ServiceSpec struct {
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:enum=ClusterIP;NodePort;LoadBalancer;ExternalName
	// +kubebuilder:default=ClusterIP
	Type corev1.ServiceType `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=18080
	Port int32 `json:"port,omitempty"`
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

type ExchangeManagerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="filesystem"
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/tmp/trino-local-file-system-exchange-manager"
	BaseDir string `json:"baseDir,omitempty"`
}

type NodePropertiesSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="production"
	Environment string `json:"environment,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=/data/trino
	DataDir string `json:"dataDir,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=/usr/lib/trino/plugin
	PluginDir string `json:"pluginDir,omitempty"`
}

type ConfigPropertiesSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/etc/trino"
	Path string `json:"path,omitempty"`
	// +kubebuilder:validation:Optional
	Https *HttpsSpec `json:"https,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="4GB"
	QueryMaxMemory string `json:"queryMaxMemory,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	AuthenticationType string `json:"authenticationType,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	MemoryHeapHeadroomPerNode string `json:"memoryHeapHeadroomPerNode,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="1GB"
	QueryMaxMemoryPerNode string `json:"queryMaxMemoryPerNode,omitempty"`
}

type HttpsSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=8443
	Port int `json:"port,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	KeystorePath string `json:"keystorePath,omitempty"`
}

type CoordinatorSpec struct {
	// +kubebuilder:validation:Optional
	RoleConfig *RoleConfigSpec `json:"roleConfig,omitempty"`

	// +kubebuilder:validation:Optional
	RoleGroups map[string]*RoleGroupCoordinatorSpec `json:"roleGroups,omitempty"`
}

type RoleConfigSpec struct {
	// +kubebuilder:validation:Optional
	JvmProperties *JvmPropertiesRoleConfigSpec `json:"jvmProperties,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigProperties *ConfigPropertiesSpec `json:"configProperties,omitempty"`
}

type JvmPropertiesRoleConfigSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="8G"
	MaxHeapSize string `json:"maxHeapSize,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="UseG1GC"
	GcMethodType string `json:"gcMethodType,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="32M"
	G1HeapRegionSize string `json:"gcHeapRegionSize,omitempty"`
}

type RoleGroupCoordinatorSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	Config *ConfigRoleGroupSpec `json:"config,omitempty"`
}

type WorkerSpec struct {
	// +kubebuilder:validation:Optional
	RoleConfig *RoleConfigSpec `json:"roleConfig,omitempty"`

	// +kubebuilder:validation:Optional
	RoleGroups map[string]*RoleGroupsWorkerSpec `json:"roleGroups,omitempty"`
}

type RoleGroupsWorkerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	Config *ConfigRoleGroupSpec `json:"config,omitempty"`
}

type ConfigRoleGroupSpec struct {
	// +kubebuilder:validation:Optional
	Image *ImageSpec `json:"image,omitempty"`

	// +kubebuilder:validation:Optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// +kubebuilder:validation:Optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`

	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	Tolerations *corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:validation:Required
	Resources *corev1.ResourceRequirements `json:"resources"`

	// +kubebuilder:validation:Optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// +kubebuilder:validation:Optional
	JvmProperties *JvmPropertiesRoleConfigSpec `json:"jvmProperties,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigProperties *ConfigPropertiesSpec `json:"configProperties,omitempty"`
}

// SetStatusCondition updates the status condition using the provided arguments.
// If the condition already exists, it updates the condition; otherwise, it appends the condition.
// If the condition status has changed, it updates the condition's LastTransitionTime.
func (r *TrinoCluster) SetStatusCondition(condition metav1.Condition) {
	r.Status.SetStatusCondition(condition)
}

// InitStatusConditions initializes the status conditions to the provided conditions.
func (r *TrinoCluster) InitStatusConditions() {
	r.Status.InitStatus(r)
	r.Status.InitStatusConditions()
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TrinoCluster is the Schema for the trinoclusters API
type TrinoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoClusterSpec `json:"spec,omitempty"`
	Status status.Status    `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TrinoClusterList contains a list of TrinoCluster
type TrinoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrinoCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrinoCluster{}, &TrinoClusterList{})
}
