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
	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/constants"
	"github.com/zncdatadev/operator-go/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultRepository      = "quay.io/zncdatadev"
	DefaultProductVersion  = "451"
	DefaultKubedoopVersion = "0.0.0-dev"
	DefaultProductName     = "trino"
)

const (
	TrinoCoordinatorRoleName       = "coordinator"
	TrinoWorkerRoleName            = "worker"
	HttpPortName                   = "http"
	HttpPort                 int32 = 8080
	HttpsPortName                  = "https"
	HttpsPort                int32 = 8443
)

const (
	DefaultTlsSecretClass = "tls"
	DefaultListenerClass  = constants.ClusterInternal
	DefaultQueryMaxMemory = "50GB"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TrinoCluster is the Schema for the trinoclusters API
type TrinoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoClusterSpec `json:"spec,omitempty"`
	Status status.Status    `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrinoClusterList contains a list of TrinoCluster
type TrinoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrinoCluster `json:"items"`
}

// TrinoSpec defines the desired state of TrinoCluster
type TrinoClusterSpec struct {

	// +kubebuilder:validation:Optional
	ClusterConfig *ClusterConfigSpec `json:"clusterConfig,omitempty"`

	// +kubebuilder:validation:Optional
	ClusterOperation *commonsv1alpha1.ClusterOperationSpec `json:"clusterOperation,omitempty"`

	// +kubebuilder:validation:Optional
	Image *ImageSpec `json:"image,omitempty"`

	// +kubebuilder:validation:Required
	Coordinators *CoordinatorsSpec `json:"coordinators"`

	// +kubebuilder:validation:Required
	Workers *WorkersSpec `json:"workers"`
}

type ClusterConfigSpec struct {

	// +kubebuilder:validation:Optional
	Authentication []AuthenticationSpec `json:"authentication,omitempty"`

	// +kubebuilder:validation:Optional
	CatalogLabelSelector *CatalogLabelSelectorSpec `json:"catalogLabelSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// TODO: to use CatalogLabelSelector instead, as it is under construction, we will use CatalogProperties for now
	CatalogProperties map[string]map[string]string `json:"catalogProperties,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="cluster-internal"
	ListenerClass constants.ListenerClass `json:"listenerClass,omitempty"`

	// +kubebuilder:validation:Optional
	Tls *TlsSpec `json:"tls,omitempty"`

	// +kubebuilder:validation:Optional
	VectorAggregatorConfigMapName string `json:"vectorAggregatorConfigMapName,omitempty"`
}

type AuthenticationSpec struct {
	// +kubebuilder:validation:Required
	AuthenticationClass string    `json:"authenticationClass"`
	Oidc                *OidcSpec `json:"oidc,omitempty"`
}

type OidcSpec struct {
	// OIDC client credentials secret. It must contain the following keys:
	//   - `CLIENT_ID`: The client ID of the OIDC client.
	//   - `CLIENT_SECRET`: The client secret of the OIDC client.
	// credentials will omit to pod environment variables.
	// +kubebuilder:validation:Required
	ClientCredentialsSecret string `json:"clientCredentialsSecret"`
	// +kubebuilder:validation:Optional
	ExtraScopes []string `json:"extraScopes,omitempty"`
}

type CatalogLabelSelectorSpec struct {
	// +kubebuilder:validation:Optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
	// +kubebuilder:validation:Optional
	MatchExpressions []metav1.LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

type TlsSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="tls"
	ServerSecretClass string `json:"serverSecretClass,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="tls"
	InternalSecretClass string `json:"internalSecretClass,omitempty"`
}

type BaseRoleSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Replicas *int32 `json:"replicas"`

	*commonsv1alpha1.OverridesSpec `json:",inline"`

	// +kubebuilder:validation:Optional
	RoleConfig *commonsv1alpha1.RoleConfigSpec `json:"roleConfig,omitempty"`

	// +kubebuilder:validation:Optional
	Config *ConfigSpec `json:"config,omitempty"`
}

type CoordinatorsSpec struct {
	// +kubebuilder:validation:Required
	RoleGroups map[string]*RoleGroupSpec `json:"roleGroups"`

	BaseRoleSpec `json:",inline"`
}

type WorkersSpec struct {
	// +kubebuilder:validation:Required
	RoleGroups map[string]*RoleGroupSpec `json:"roleGroups"`

	BaseRoleSpec `json:",inline"`
}

type RoleGroupSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Replicas *int32 `json:"replicas,omitempty"`

	Config *ConfigSpec `json:"config,omitempty"`

	*commonsv1alpha1.OverridesSpec `json:",inline"`
}

type ConfigSpec struct {
	*commonsv1alpha1.RoleGroupConfigSpec `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="5GB"
	QueryMaxMemory string `json:"queryMaxMemory,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="1000MB"
	QueryMaxMemoryPerNode string `json:"queryMaxMemoryPerNode,omitempty"`
}

type LoggingSpec struct {
	// +kubebuilder:validation:Optional
	Containers map[string]commonsv1alpha1.LoggingConfigSpec `json:"containers,omitempty"`

	// +kubebuilder:validation:Optional
	EnableVectorAgent bool `json:"enableVectorAgent,omitempty"`
}

type ConfigOverridesSpec struct {
	Node            map[string]string `json:"node.properties,omitempty"`
	Jvm             string            `json:"jvm.config,omitempty"`
	Config          map[string]string `json:"config.properties,omitempty"`
	Log             map[string]string `json:"log.properties,omitempty"`
	ExchangeManager map[string]string `json:"exchange-manager.properties,omitempty"`
}

type ImageSpec struct {
	// +kubebuilder:validation:Optional
	Custom string `json:"custom,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=quay.io/zncdatadev
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0-dev"
	KubedoopVersion string `json:"kubedoopVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="451"
	ProductVersion string `json:"productVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=IfNotPresent
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="trino"
	PullSecretName string `json:"pullSecretName,omitempty"`
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

func init() {
	SchemeBuilder.Register(&TrinoCluster{}, &TrinoClusterList{})
}
