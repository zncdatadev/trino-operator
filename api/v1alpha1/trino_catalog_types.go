package v1alpha1

import (
	s3v1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/s3/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TrinoCatalog is the Schema for the TrinoCatalog API
type TrinoCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoSpec     `json:"spec,omitempty"`
	Status status.Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TrinoCatalogList contains a list of TrinoCatalog
type TrinoCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrinoCatalog `json:"items"`
}

// TrinoCatalogSpec defines the desired state of TrinoCatalog
type TrinoCatalogSpec struct {
	// List of connectors in the catalog
	// +kubebuilder:validation:required
	Connectors []ConnectorSpec `json:"connectors"`

	// The configOverrides allow overriding arbitrary Trino settings. For example, for Hive you could add hive.metastore.username: trino.
	// +kubebuilder:validation:Optional
	ConfigOverrides map[string]string `json:"configOverrides,omitempty"`
}

type ConnectorSpec struct {
	// +kubebuilder:validation:optional
	Generic *GenericConnectorSpec `json:"generic,omitempty"`

	// +kubebuilder:validation:optional
	Hive *HiveConnectorSpec `json:"hive,omitempty"`

	// +kubebuilder:validation:optional
	IceBerg *IcebergConnectorSpec `json:"iceberg,omitempty"`

	// +kubebuilder:validation:optional
	Tpcds *TpcdsConnectorSpec `json:"tpcds,omitempty"`

	// +kubebuilder:validation:optional
	Tpch *TpchConnectorSpec `json:"tpch,omitempty"`
}

type GenericConnectorSpec struct {
	// +kubebuilder:validation:required
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	Properites map[string]string `json:"properties,omitempty"`
}

type HiveConnectorSpec struct {
	// +kubebuilder:validation:required
	Metastore *MetastoreConnectionSpec `json:"metastore,omitempty"`

	// +kubebuilder:validation:optional
	S3 *s3v1alpha1.S3BucketSpec `json:"s3,omitempty"`

	// +kubebuilder:validation:optional
	Hdfs *HdfsConnectionSpec `json:"hdfs,omitempty"`
}

type IcebergConnectorSpec struct {
	// +kubebuilder:validation:required
	Metastore *MetastoreConnectionSpec `json:"metastore,omitempty"`

	// +kubebuilder:validation:optional
	S3 *s3v1alpha1.S3BucketSpec `json:"s3,omitempty"`

	// +kubebuilder:validation:optional
	Hdfs *HdfsConnectionSpec `json:"hdfs,omitempty"`
}

type TpcdsConnectorSpec struct {
}

type TpchConnectorSpec struct {
}

type PropertiesSpec struct {
	// +kubebuilder:validation:optional
	Value string `json:"value,omitempty"`

	// +kubebuilder:validation:optional
	ValueFromConfiguration *ValueFromConfigurationSpec `json:"valueFromConfiguration,omitempty"`
}

type ValueFromConfigurationSpec struct {
	// +kubebuilder:validation:rquired
	// +kubebuilder:default=configmap
	// +kubebuilder:validation:Enum=configmap;secret
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:required
	Name string `json:"name"`

	// +kubebuilder:validation:required
	Key string `json:"key,omitempty"`
}

type MetastoreConnectionSpec struct {
	// +kubebuilder:validation:required
	ConfigMap string `json:"configMap,omitempty"`
}

type HdfsConnectionSpec struct {
	// +kubebuilder:validation:required
	ConfigMap string `json:"configMap,omitempty"`
}
