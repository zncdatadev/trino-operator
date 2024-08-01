/*
Copyright 2023 zncdatadev.

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
	"github.com/zncdatadev/operator-go/pkg/status"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const (
	NodePropertiesFileName            = "node.properties"
	JvmConfigFileName                 = "jvm.config"
	ConfigPropertiesFileName          = "config.properties"
	LogPropertiesFileName             = "log.properties"
	ExchangeManagerPropertiesFileName = "exchange-manager.properties"
	VectorYamlName                    = "vector.yaml"
)

const (
	// resource
	CpuMin      = "1"
	CpuMax      = "1.5"
	MemoryLimit = "1.5Gi"

	//service
	ServiceType = "ClusterIP"
	ServicePort = 18080

	//exchange manager
	ExchangeManagerName    = "filesystem"
	ExchangeManagerBaseDir = "/tmp/TrinoCluster-local-file-system-exchange-manager"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TrinoCluster is the Schema for the trinoclusters API
type TrinoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoSpec     `json:"spec,omitempty"`
	Status status.Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TrinoClusterList contains a list of TrinoCluster
type TrinoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrinoCluster `json:"items"`
}

// TrinoSpec defines the desired state of TrinoCluster
type TrinoSpec struct {
	// +kubebuilder:validation:Required
	Image *ImageSpec `json:"image"`

	// +kubebuilder:validation:Required
	Coordinator *CoordinatorSpec `json:"coordinator"`

	// +kubebuilder:validation:Required
	Worker *WorkerSpec `json:"worker"`

	// +kubebuilder:validation:Optional
	ClusterConfig *ClusterConfigSpec `json:"clusterConfig,omitempty"`

	// +kubebuilder:validation:Optional
	ClusterOperation *ClusterOperationSpec `json:"clusterOperation,omitempty"`
}

type ClusterOperationSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	ReconciliationPaused bool `json:"reconciliationPaused,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Stopped bool `json:"stopped,omitempty"`
}

type ClusterConfigSpec struct {
	// +kubebuilder:validation:Optional
	VectorAggregatorConfigMapName string `json:"vectorAggregatorConfigMapName,omitempty"`
	// +kubebuilder:validation:Optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// +kubebuilder:validation:Optional
	Catalogs map[string]string `json:"catalogs,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default:=true
	ClusterMode bool `json:"clusterMode"`
}

type CoordinatorSpec struct {
	// +kubebuilder:validation:Optional
	Config *ConfigSpec `json:"config,omitempty"`

	// +kubebuilder:validation:Optional
	RoleGroups map[string]*RoleGroupSpec `json:"roleGroups,omitempty"`

	// +kubebuilder:validation:Optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// +kubebuilder:validation:Optional
	CommandArgsOverrides []string `json:"commandArgsOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigOverrides *ConfigOverridesSpec `json:"configOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	EnvOverrides map[string]string `json:"envOverrides,omitempty"`

	//// +kubebuilder:validation:Optional
	//PodOverride corev1.PodSpec `json:"podOverride,omitempty"`
}

type WorkerSpec struct {
	// +kubebuilder:validation:Optional
	Config *ConfigSpec `json:"config,omitempty"`

	RoleGroups map[string]*RoleGroupSpec `json:"roleGroups,omitempty"`

	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// +kubebuilder:validation:Optional
	CommandArgsOverrides []string `json:"commandArgsOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigOverrides *ConfigOverridesSpec `json:"configOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	EnvOverrides map[string]string `json:"envOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	PodOverride *corev1.PodTemplateSpec `json:"podOverride,omitempty"`
}

type RoleGroupSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	Config *ConfigSpec `json:"config,omitempty"`

	// +kubebuilder:validation:Optional
	CommandArgsOverrides []string `json:"commandArgsOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigOverrides *ConfigOverridesSpec `json:"configOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	EnvOverrides map[string]string `json:"envOverrides,omitempty"`

	// +kubebuilder:validation:Optional
	PodOverride *corev1.PodTemplateSpec `json:"podOverride,omitempty"`
}

type ConfigSpec struct {
	// +kubebuilder:validation:Optional
	Resources *ResourcesSpec `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity"`

	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations"`

	// +kubebuilder:validation:Optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// Use time.ParseDuration to parse the string
	// +kubebuilder:validation:Optional
	GracefulShutdownTimeout *string `json:"gracefulShutdownTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	NodeProperties *NodePropertiesSpec `json:"nodeProperties,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigProperties *ConfigPropertiesSpec `json:"configProperties,omitempty"`

	// +kubebuilder:validation:Optional
	JvmProperties *JvmPropertiesRoleConfigSpec `json:"jvmProperties,omitempty"`

	// +kubebuilder:validation:Optional
	ExchangeManager *ExchangeManagerSpec `json:"exchangeManager,omitempty"`

	// +kubebuilder:validation:Optional
	Logging *ContainerLoggingSpec `json:"logging,omitempty"`
}

type ConfigOverridesSpec struct {
	Node            map[string]string `json:"node.properties,omitempty"`
	Jvm             string            `json:"jvm.config,omitempty"`
	Config          map[string]string `json:"config.properties,omitempty"`
	Log             map[string]string `json:"log.properties,omitempty"`
	ExchangeManager map[string]string `json:"exchange-manager.properties,omitempty"`
}

type PodDisruptionBudgetSpec struct {
	// +kubebuilder:validation:Optional
	MinAvailable int32 `json:"minAvailable,omitempty"`

	// +kubebuilder:validation:Optional
	MaxUnavailable int32 `json:"maxUnavailable,omitempty"`
}

type ImageSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=trinodb/TrinoCluster
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
	// +kubebuilder:default:="TrinoCluster.example.com"
	Host string `json:"host,omitempty"`
}

type ExchangeManagerSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="filesystem"
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/tmp/TrinoCluster-local-file-system-exchange-manager"
	BaseDir string `json:"baseDir,omitempty"`
}

type NodePropertiesSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="production"
	Environment string `json:"environment,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=/data/TrinoCluster
	DataDir string `json:"dataDir,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=/usr/lib/TrinoCluster/plugin
	PluginDir string `json:"pluginDir,omitempty"`
}

type ConfigPropertiesSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/etc/TrinoCluster"
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

func init() {
	SchemeBuilder.Register(&TrinoCluster{}, &TrinoClusterList{})
}
func (r *TrinoCluster) GetNameWithSuffix(suffix string) string {
	return r.GetName() + "-" + suffix
}
