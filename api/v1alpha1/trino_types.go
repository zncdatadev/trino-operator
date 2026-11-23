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
	operatorcommon "github.com/zncdatadev/operator-go/pkg/common"
	"github.com/zncdatadev/operator-go/pkg/listener"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultRepository     = "quay.io/zncdatadev"
	DefaultProductVersion = "451"
	DefaultProductName    = "trino"
)

const (
	TrinoCoordinatorRoleName       = "coordinator"
	TrinoWorkerRoleName            = "worker"
	HttpPortName                   = "http"
	HttpPort                 int32 = 8080
	HttpsPortName                  = "https"
	HttpsPort                int32 = 8443
	MetricsPortName                = "metrics"
	MetricsPort              int32 = 8081
)

const (
	DefaultTlsSecretClass = "tls"
	DefaultListenerClass  = listener.ListenerClassClusterInternal
	DefaultQueryMaxMemory = "50GB"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TrinoCluster is the Schema for the trinoclusters API
type TrinoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoClusterSpec   `json:"spec,omitempty"`
	Status TrinoClusterStatus `json:"status,omitempty"`
}

// TrinoClusterStatus combines the status managed by the GenericReconciler with the legacy
// fields exposed by this CRD. The legacy fields stay available during the framework migration so
// an operator upgrade does not prune status written by an older release.
type TrinoClusterStatus struct {
	commonsv1alpha1.GenericClusterStatus `json:",inline"`

	URLs       []StatusURL `json:"urls,omitempty"`
	Generation int64       `json:"generation,omitempty"`
	Name       string      `json:"name,omitempty"`
	Type       string      `json:"type,omitempty"`
}

// StatusURL is the legacy named URL shape formerly supplied by operator-go/pkg/status.
type StatusURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// DeepCopyInto is implemented by hand so the existing generated TrinoCluster deepcopy remains
// usable while generated files are refreshed together after the source migration. controller-gen
// recognizes this method and will not emit a duplicate implementation.
func (in *TrinoClusterStatus) DeepCopyInto(out *TrinoClusterStatus) {
	*out = *in
	in.GenericClusterStatus.DeepCopyInto(&out.GenericClusterStatus)
	if in.URLs != nil {
		out.URLs = append([]StatusURL(nil), in.URLs...)
	}
}

var (
	_ operatorcommon.ClusterInterface     = (*TrinoCluster)(nil)
	_ reconciler.VectorAggregatorProvider = (*TrinoCluster)(nil)
)

// GetSpec projects the typed Trino roles onto the generic framework shape without applying
// defaults. In particular, role group replica pointers pass through unchanged so an explicit zero
// remains distinguishable from an omitted value.
func (t *TrinoCluster) GetSpec() *commonsv1alpha1.GenericClusterSpec {
	return t.Spec.ToGenericSpec()
}

// GetStatus returns the status storage that the GenericReconciler updates in place.
func (t *TrinoCluster) GetStatus() *commonsv1alpha1.GenericClusterStatus {
	return &t.Status.GenericClusterStatus
}

// VectorAggregatorConfigMapName implements reconciler.VectorAggregatorProvider.
func (t *TrinoCluster) VectorAggregatorConfigMapName() string {
	if t.Spec.ClusterConfig == nil {
		return ""
	}
	return t.Spec.ClusterConfig.VectorAggregatorConfigMapName
}

// ToGenericSpec adapts the existing typed Trino API to the GenericReconciler contract. Singular
// role keys preserve the resource names and component labels emitted by the legacy controller.
func (s *TrinoClusterSpec) ToGenericSpec() *commonsv1alpha1.GenericClusterSpec {
	result := &commonsv1alpha1.GenericClusterSpec{
		ClusterOperation: s.ClusterOperation,
		Image:            adaptImage(s.Image),
	}

	roles := make(map[string]commonsv1alpha1.RoleSpec, 2)
	if s.Coordinators != nil {
		roles[TrinoCoordinatorRoleName] = adaptRoleSpec(&s.Coordinators.BaseRoleSpec, s.Coordinators.RoleGroups)
	}
	if s.Workers != nil {
		roles[TrinoWorkerRoleName] = adaptRoleSpec(&s.Workers.BaseRoleSpec, s.Workers.RoleGroups)
	}
	result.Roles = roles

	return result
}

func adaptImage(image *ImageSpec) *commonsv1alpha1.ImageSpec {
	if image == nil {
		return nil
	}
	return &commonsv1alpha1.ImageSpec{
		Custom:          image.Custom,
		Repo:            image.Repo,
		ProductVersion:  image.ProductVersion,
		KubedoopVersion: image.KubedoopVersion,
		PullPolicy:      image.PullPolicy,
		PullSecretName:  image.PullSecretName,
	}
}

func adaptRoleSpec(
	base *BaseRoleSpec,
	roleGroups map[string]*RoleGroupSpec,
) commonsv1alpha1.RoleSpec {
	adapted := commonsv1alpha1.RoleSpec{
		RoleConfig: base.RoleConfig,
	}
	if base.Config != nil {
		adapted.Config = base.Config.RoleGroupConfigSpec
	}
	copyOverridesToRole(&adapted, base.OverridesSpec)

	if roleGroups != nil {
		adapted.RoleGroups = make(map[string]commonsv1alpha1.RoleGroupSpec, len(roleGroups))
		for name, roleGroup := range roleGroups {
			if roleGroup == nil {
				continue
			}
			adapted.RoleGroups[name] = adaptRoleGroupSpec(roleGroup)
		}
	}

	return adapted
}

func adaptRoleGroupSpec(roleGroup *RoleGroupSpec) commonsv1alpha1.RoleGroupSpec {
	adapted := commonsv1alpha1.RoleGroupSpec{
		Replicas: roleGroup.Replicas,
	}
	if roleGroup.Config != nil {
		adapted.Config = roleGroup.Config.RoleGroupConfigSpec
	}
	copyOverridesToRoleGroup(&adapted, roleGroup.OverridesSpec)
	return adapted
}

func copyOverridesToRole(target *commonsv1alpha1.RoleSpec, overrides *commonsv1alpha1.OverridesSpec) {
	if overrides == nil {
		return
	}
	target.ConfigOverrides = overrides.ConfigOverrides
	target.EnvOverrides = overrides.EnvOverrides
	target.CliOverrides = overrides.CliOverrides
	target.PodOverrides = overrides.PodOverrides
}

func copyOverridesToRoleGroup(target *commonsv1alpha1.RoleGroupSpec, overrides *commonsv1alpha1.OverridesSpec) {
	if overrides == nil {
		return
	}
	target.ConfigOverrides = overrides.ConfigOverrides
	target.EnvOverrides = overrides.EnvOverrides
	target.CliOverrides = overrides.CliOverrides
	target.PodOverrides = overrides.PodOverrides
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
	// +default:value={"repo": "quay.io/zncdatadev", "pullPolicy": "IfNotPresent"}
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
	ListenerClass listener.ListenerClass `json:"listenerClass,omitempty"`

	// +kubebuilder:validation:Optional
	Tls *TlsSpec `json:"tls,omitempty"`

	// +kubebuilder:validation:Optional
	VectorAggregatorConfigMapName string `json:"vectorAggregatorConfigMapName,omitempty"`
}

type AuthenticationSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AuthenticationClass string    `json:"authenticationClass"`
	Oidc                *OidcSpec `json:"oidc,omitempty"`
}

type OidcSpec struct {
	// OIDC client credentials secret. It must contain the following keys:
	//   - `CLIENT_ID`: The client ID of the OIDC client.
	//   - `CLIENT_SECRET`: The client secret of the OIDC client.
	// credentials will omit to pod environment variables.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
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
	QueryMaxMemory string `json:"queryMaxMemory,omitempty"`

	// +kubebuilder:validation:Optional
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
	Repo string `json:"repo,omitempty"`

	// +kubebuilder:validation:Optional
	KubedoopVersion string `json:"kubedoopVersion,omitempty"`

	// +kubebuilder:validation:Optional
	ProductVersion string `json:"productVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=IfNotPresent
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`

	// +kubebuilder:validation:Optional
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
