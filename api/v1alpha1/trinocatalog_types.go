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
	s3v1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/s3/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	Properties *PropertiesSpec `json:"properties,omitempty"`
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

// TrinoCatalogStatus defines the observed state of TrinoCatalog
type TrinoCatalogStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TrinoCatalog is the Schema for the trinocatalogs API
type TrinoCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrinoCatalogSpec   `json:"spec,omitempty"`
	Status TrinoCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrinoCatalogList contains a list of TrinoCatalog
type TrinoCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrinoCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrinoCatalog{}, &TrinoCatalogList{})
}
