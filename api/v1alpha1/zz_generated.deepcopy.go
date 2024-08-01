//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	s3v1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/s3/v1alpha1"
	"k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CPUResource) DeepCopyInto(out *CPUResource) {
	*out = *in
	if in.Max != nil {
		in, out := &in.Max, &out.Max
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Min != nil {
		in, out := &in.Min, &out.Min
		x := (*in).DeepCopy()
		*out = &x
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CPUResource.
func (in *CPUResource) DeepCopy() *CPUResource {
	if in == nil {
		return nil
	}
	out := new(CPUResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterConfigSpec) DeepCopyInto(out *ClusterConfigSpec) {
	*out = *in
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Ingress != nil {
		in, out := &in.Ingress, &out.Ingress
		*out = new(IngressSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Catalogs != nil {
		in, out := &in.Catalogs, &out.Catalogs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterConfigSpec.
func (in *ClusterConfigSpec) DeepCopy() *ClusterConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterOperationSpec) DeepCopyInto(out *ClusterOperationSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterOperationSpec.
func (in *ClusterOperationSpec) DeepCopy() *ClusterOperationSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterOperationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigOverridesSpec) DeepCopyInto(out *ConfigOverridesSpec) {
	*out = *in
	if in.Node != nil {
		in, out := &in.Node, &out.Node
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Log != nil {
		in, out := &in.Log, &out.Log
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ExchangeManager != nil {
		in, out := &in.ExchangeManager, &out.ExchangeManager
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigOverridesSpec.
func (in *ConfigOverridesSpec) DeepCopy() *ConfigOverridesSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigOverridesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigPropertiesSpec) DeepCopyInto(out *ConfigPropertiesSpec) {
	*out = *in
	if in.Https != nil {
		in, out := &in.Https, &out.Https
		*out = new(HttpsSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigPropertiesSpec.
func (in *ConfigPropertiesSpec) DeepCopy() *ConfigPropertiesSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigPropertiesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigSpec) DeepCopyInto(out *ConfigSpec) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(ResourcesSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodDisruptionBudget != nil {
		in, out := &in.PodDisruptionBudget, &out.PodDisruptionBudget
		*out = new(PodDisruptionBudgetSpec)
		**out = **in
	}
	if in.GracefulShutdownTimeout != nil {
		in, out := &in.GracefulShutdownTimeout, &out.GracefulShutdownTimeout
		*out = new(string)
		**out = **in
	}
	if in.NodeProperties != nil {
		in, out := &in.NodeProperties, &out.NodeProperties
		*out = new(NodePropertiesSpec)
		**out = **in
	}
	if in.ConfigProperties != nil {
		in, out := &in.ConfigProperties, &out.ConfigProperties
		*out = new(ConfigPropertiesSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.JvmProperties != nil {
		in, out := &in.JvmProperties, &out.JvmProperties
		*out = new(JvmPropertiesRoleConfigSpec)
		**out = **in
	}
	if in.ExchangeManager != nil {
		in, out := &in.ExchangeManager, &out.ExchangeManager
		*out = new(ExchangeManagerSpec)
		**out = **in
	}
	if in.Logging != nil {
		in, out := &in.Logging, &out.Logging
		*out = new(ContainerLoggingSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigSpec.
func (in *ConfigSpec) DeepCopy() *ConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConnectorSpec) DeepCopyInto(out *ConnectorSpec) {
	*out = *in
	if in.Generic != nil {
		in, out := &in.Generic, &out.Generic
		*out = new(GenericConnectorSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Hive != nil {
		in, out := &in.Hive, &out.Hive
		*out = new(HiveConnectorSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.IceBerg != nil {
		in, out := &in.IceBerg, &out.IceBerg
		*out = new(IcebergConnectorSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Tpcds != nil {
		in, out := &in.Tpcds, &out.Tpcds
		*out = new(TpcdsConnectorSpec)
		**out = **in
	}
	if in.Tpch != nil {
		in, out := &in.Tpch, &out.Tpch
		*out = new(TpchConnectorSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConnectorSpec.
func (in *ConnectorSpec) DeepCopy() *ConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(ConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerLoggingSpec) DeepCopyInto(out *ContainerLoggingSpec) {
	*out = *in
	if in.Trino != nil {
		in, out := &in.Trino, &out.Trino
		*out = new(LoggingConfigSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerLoggingSpec.
func (in *ContainerLoggingSpec) DeepCopy() *ContainerLoggingSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerLoggingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoordinatorSpec) DeepCopyInto(out *CoordinatorSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(ConfigSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.RoleGroups != nil {
		in, out := &in.RoleGroups, &out.RoleGroups
		*out = make(map[string]*RoleGroupSpec, len(*in))
		for key, val := range *in {
			var outVal *RoleGroupSpec
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = new(RoleGroupSpec)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.PodDisruptionBudget != nil {
		in, out := &in.PodDisruptionBudget, &out.PodDisruptionBudget
		*out = new(PodDisruptionBudgetSpec)
		**out = **in
	}
	if in.CommandArgsOverrides != nil {
		in, out := &in.CommandArgsOverrides, &out.CommandArgsOverrides
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ConfigOverrides != nil {
		in, out := &in.ConfigOverrides, &out.ConfigOverrides
		*out = new(ConfigOverridesSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.EnvOverrides != nil {
		in, out := &in.EnvOverrides, &out.EnvOverrides
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoordinatorSpec.
func (in *CoordinatorSpec) DeepCopy() *CoordinatorSpec {
	if in == nil {
		return nil
	}
	out := new(CoordinatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExchangeManagerSpec) DeepCopyInto(out *ExchangeManagerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExchangeManagerSpec.
func (in *ExchangeManagerSpec) DeepCopy() *ExchangeManagerSpec {
	if in == nil {
		return nil
	}
	out := new(ExchangeManagerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericConnectorSpec) DeepCopyInto(out *GenericConnectorSpec) {
	*out = *in
	if in.Properites != nil {
		in, out := &in.Properites, &out.Properites
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericConnectorSpec.
func (in *GenericConnectorSpec) DeepCopy() *GenericConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(GenericConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HdfsConnectionSpec) DeepCopyInto(out *HdfsConnectionSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HdfsConnectionSpec.
func (in *HdfsConnectionSpec) DeepCopy() *HdfsConnectionSpec {
	if in == nil {
		return nil
	}
	out := new(HdfsConnectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HiveConnectorSpec) DeepCopyInto(out *HiveConnectorSpec) {
	*out = *in
	if in.Metastore != nil {
		in, out := &in.Metastore, &out.Metastore
		*out = new(MetastoreConnectionSpec)
		**out = **in
	}
	if in.S3 != nil {
		in, out := &in.S3, &out.S3
		*out = new(s3v1alpha1.S3BucketSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Hdfs != nil {
		in, out := &in.Hdfs, &out.Hdfs
		*out = new(HdfsConnectionSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HiveConnectorSpec.
func (in *HiveConnectorSpec) DeepCopy() *HiveConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(HiveConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HttpsSpec) DeepCopyInto(out *HttpsSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HttpsSpec.
func (in *HttpsSpec) DeepCopy() *HttpsSpec {
	if in == nil {
		return nil
	}
	out := new(HttpsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IcebergConnectorSpec) DeepCopyInto(out *IcebergConnectorSpec) {
	*out = *in
	if in.Metastore != nil {
		in, out := &in.Metastore, &out.Metastore
		*out = new(MetastoreConnectionSpec)
		**out = **in
	}
	if in.S3 != nil {
		in, out := &in.S3, &out.S3
		*out = new(s3v1alpha1.S3BucketSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Hdfs != nil {
		in, out := &in.Hdfs, &out.Hdfs
		*out = new(HdfsConnectionSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IcebergConnectorSpec.
func (in *IcebergConnectorSpec) DeepCopy() *IcebergConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(IcebergConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSpec.
func (in *ImageSpec) DeepCopy() *ImageSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressSpec) DeepCopyInto(out *IngressSpec) {
	*out = *in
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = new(networkingv1.IngressTLS)
		(*in).DeepCopyInto(*out)
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressSpec.
func (in *IngressSpec) DeepCopy() *IngressSpec {
	if in == nil {
		return nil
	}
	out := new(IngressSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JvmPropertiesRoleConfigSpec) DeepCopyInto(out *JvmPropertiesRoleConfigSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JvmPropertiesRoleConfigSpec.
func (in *JvmPropertiesRoleConfigSpec) DeepCopy() *JvmPropertiesRoleConfigSpec {
	if in == nil {
		return nil
	}
	out := new(JvmPropertiesRoleConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogLevelSpec) DeepCopyInto(out *LogLevelSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogLevelSpec.
func (in *LogLevelSpec) DeepCopy() *LogLevelSpec {
	if in == nil {
		return nil
	}
	out := new(LogLevelSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoggingConfigSpec) DeepCopyInto(out *LoggingConfigSpec) {
	*out = *in
	if in.Loggers != nil {
		in, out := &in.Loggers, &out.Loggers
		*out = make(map[string]*LogLevelSpec, len(*in))
		for key, val := range *in {
			var outVal *LogLevelSpec
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = new(LogLevelSpec)
				**out = **in
			}
			(*out)[key] = outVal
		}
	}
	if in.Console != nil {
		in, out := &in.Console, &out.Console
		*out = new(LogLevelSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoggingConfigSpec.
func (in *LoggingConfigSpec) DeepCopy() *LoggingConfigSpec {
	if in == nil {
		return nil
	}
	out := new(LoggingConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MemoryResource) DeepCopyInto(out *MemoryResource) {
	*out = *in
	if in.Limit != nil {
		in, out := &in.Limit, &out.Limit
		x := (*in).DeepCopy()
		*out = &x
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MemoryResource.
func (in *MemoryResource) DeepCopy() *MemoryResource {
	if in == nil {
		return nil
	}
	out := new(MemoryResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetastoreConnectionSpec) DeepCopyInto(out *MetastoreConnectionSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetastoreConnectionSpec.
func (in *MetastoreConnectionSpec) DeepCopy() *MetastoreConnectionSpec {
	if in == nil {
		return nil
	}
	out := new(MetastoreConnectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePropertiesSpec) DeepCopyInto(out *NodePropertiesSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePropertiesSpec.
func (in *NodePropertiesSpec) DeepCopy() *NodePropertiesSpec {
	if in == nil {
		return nil
	}
	out := new(NodePropertiesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodDisruptionBudgetSpec) DeepCopyInto(out *PodDisruptionBudgetSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodDisruptionBudgetSpec.
func (in *PodDisruptionBudgetSpec) DeepCopy() *PodDisruptionBudgetSpec {
	if in == nil {
		return nil
	}
	out := new(PodDisruptionBudgetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PropertiesSpec) DeepCopyInto(out *PropertiesSpec) {
	*out = *in
	if in.ValueFromConfiguration != nil {
		in, out := &in.ValueFromConfiguration, &out.ValueFromConfiguration
		*out = new(ValueFromConfigurationSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PropertiesSpec.
func (in *PropertiesSpec) DeepCopy() *PropertiesSpec {
	if in == nil {
		return nil
	}
	out := new(PropertiesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcesSpec) DeepCopyInto(out *ResourcesSpec) {
	*out = *in
	if in.CPU != nil {
		in, out := &in.CPU, &out.CPU
		*out = new(CPUResource)
		(*in).DeepCopyInto(*out)
	}
	if in.Memory != nil {
		in, out := &in.Memory, &out.Memory
		*out = new(MemoryResource)
		(*in).DeepCopyInto(*out)
	}
	if in.Storage != nil {
		in, out := &in.Storage, &out.Storage
		*out = new(StorageResource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourcesSpec.
func (in *ResourcesSpec) DeepCopy() *ResourcesSpec {
	if in == nil {
		return nil
	}
	out := new(ResourcesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RoleGroupSpec) DeepCopyInto(out *RoleGroupSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(ConfigSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CommandArgsOverrides != nil {
		in, out := &in.CommandArgsOverrides, &out.CommandArgsOverrides
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ConfigOverrides != nil {
		in, out := &in.ConfigOverrides, &out.ConfigOverrides
		*out = new(ConfigOverridesSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.EnvOverrides != nil {
		in, out := &in.EnvOverrides, &out.EnvOverrides
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodOverride != nil {
		in, out := &in.PodOverride, &out.PodOverride
		*out = new(v1.PodTemplateSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoleGroupSpec.
func (in *RoleGroupSpec) DeepCopy() *RoleGroupSpec {
	if in == nil {
		return nil
	}
	out := new(RoleGroupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceSpec) DeepCopyInto(out *ServiceSpec) {
	*out = *in
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceSpec.
func (in *ServiceSpec) DeepCopy() *ServiceSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageResource) DeepCopyInto(out *StorageResource) {
	*out = *in
	out.Capacity = in.Capacity.DeepCopy()
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageResource.
func (in *StorageResource) DeepCopy() *StorageResource {
	if in == nil {
		return nil
	}
	out := new(StorageResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageResourceSpec) DeepCopyInto(out *StorageResourceSpec) {
	*out = *in
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = new(StorageResource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageResourceSpec.
func (in *StorageResourceSpec) DeepCopy() *StorageResourceSpec {
	if in == nil {
		return nil
	}
	out := new(StorageResourceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TpcdsConnectorSpec) DeepCopyInto(out *TpcdsConnectorSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TpcdsConnectorSpec.
func (in *TpcdsConnectorSpec) DeepCopy() *TpcdsConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(TpcdsConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TpchConnectorSpec) DeepCopyInto(out *TpchConnectorSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TpchConnectorSpec.
func (in *TpchConnectorSpec) DeepCopy() *TpchConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(TpchConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoCatalog) DeepCopyInto(out *TrinoCatalog) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoCatalog.
func (in *TrinoCatalog) DeepCopy() *TrinoCatalog {
	if in == nil {
		return nil
	}
	out := new(TrinoCatalog)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TrinoCatalog) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoCatalogList) DeepCopyInto(out *TrinoCatalogList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TrinoCatalog, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoCatalogList.
func (in *TrinoCatalogList) DeepCopy() *TrinoCatalogList {
	if in == nil {
		return nil
	}
	out := new(TrinoCatalogList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TrinoCatalogList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoCatalogSpec) DeepCopyInto(out *TrinoCatalogSpec) {
	*out = *in
	if in.Connectors != nil {
		in, out := &in.Connectors, &out.Connectors
		*out = make([]ConnectorSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ConfigOverrides != nil {
		in, out := &in.ConfigOverrides, &out.ConfigOverrides
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoCatalogSpec.
func (in *TrinoCatalogSpec) DeepCopy() *TrinoCatalogSpec {
	if in == nil {
		return nil
	}
	out := new(TrinoCatalogSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoCluster) DeepCopyInto(out *TrinoCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoCluster.
func (in *TrinoCluster) DeepCopy() *TrinoCluster {
	if in == nil {
		return nil
	}
	out := new(TrinoCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TrinoCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoClusterList) DeepCopyInto(out *TrinoClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TrinoCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoClusterList.
func (in *TrinoClusterList) DeepCopy() *TrinoClusterList {
	if in == nil {
		return nil
	}
	out := new(TrinoClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TrinoClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrinoSpec) DeepCopyInto(out *TrinoSpec) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(ImageSpec)
		**out = **in
	}
	if in.Coordinator != nil {
		in, out := &in.Coordinator, &out.Coordinator
		*out = new(CoordinatorSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Worker != nil {
		in, out := &in.Worker, &out.Worker
		*out = new(WorkerSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ClusterConfig != nil {
		in, out := &in.ClusterConfig, &out.ClusterConfig
		*out = new(ClusterConfigSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ClusterOperation != nil {
		in, out := &in.ClusterOperation, &out.ClusterOperation
		*out = new(ClusterOperationSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrinoSpec.
func (in *TrinoSpec) DeepCopy() *TrinoSpec {
	if in == nil {
		return nil
	}
	out := new(TrinoSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValueFromConfigurationSpec) DeepCopyInto(out *ValueFromConfigurationSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValueFromConfigurationSpec.
func (in *ValueFromConfigurationSpec) DeepCopy() *ValueFromConfigurationSpec {
	if in == nil {
		return nil
	}
	out := new(ValueFromConfigurationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkerSpec) DeepCopyInto(out *WorkerSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(ConfigSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.RoleGroups != nil {
		in, out := &in.RoleGroups, &out.RoleGroups
		*out = make(map[string]*RoleGroupSpec, len(*in))
		for key, val := range *in {
			var outVal *RoleGroupSpec
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = new(RoleGroupSpec)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.PodDisruptionBudget != nil {
		in, out := &in.PodDisruptionBudget, &out.PodDisruptionBudget
		*out = new(PodDisruptionBudgetSpec)
		**out = **in
	}
	if in.CommandArgsOverrides != nil {
		in, out := &in.CommandArgsOverrides, &out.CommandArgsOverrides
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ConfigOverrides != nil {
		in, out := &in.ConfigOverrides, &out.ConfigOverrides
		*out = new(ConfigOverridesSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.EnvOverrides != nil {
		in, out := &in.EnvOverrides, &out.EnvOverrides
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodOverride != nil {
		in, out := &in.PodOverride, &out.PodOverride
		*out = new(v1.PodTemplateSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkerSpec.
func (in *WorkerSpec) DeepCopy() *WorkerSpec {
	if in == nil {
		return nil
	}
	out := new(WorkerSpec)
	in.DeepCopyInto(out)
	return out
}
