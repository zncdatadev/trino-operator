package common

import (
	"github.com/zncdatadev/operator-go/pkg/builder"
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func NewTrinoContainerBuilder(
	image string,
	imagePullPolicy corev1.PullPolicy,
	resourceSpec *trinov1alpha1.ResourcesSpec,
	component ContainerComponent) *TrinoContainerBuilder {
	return &TrinoContainerBuilder{
		ContainerBuilder: NewContainerBuilder(image, imagePullPolicy),
		resourceSpec:     resourceSpec,
		component:        component,
	}
}

type Types interface {
	ContainerName
	ContainerPorts
	ResourceRequirements
	VolumeMount
	Command
	CommandArgs
}

var _ Types = &TrinoContainerBuilder{}

type TrinoContainerBuilder struct {
	*ContainerBuilder
	resourceSpec *trinov1alpha1.ResourcesSpec
	component    ContainerComponent
}

func (c *TrinoContainerBuilder) Command() []string {
	return []string{
		"/bin/bash",
		"-x",
		"-euo",
		"pipefail",
		"-c",
	}
}

func (c *TrinoContainerBuilder) CommandArgs() []string {
	trinoEntrypointScript := `
     set -xeuo pipefail
     launcher_opts=(--etc-dir /zncdata/config)
     if ! grep -s -q 'node.id' /zncdata/config/node.properties; then
       launcher_opts+=("-Dnode.id=${HOSTNAME}")
     fi
     exec /usr/lib/trino/bin/launcher run "${launcher_opts[@]}" "$@"
`
	script, err := builder.LogProviderCommand(trinoEntrypointScript)
	if err != nil {
		panic(err)
	}
	return script
}

func (c *TrinoContainerBuilder) ContainerName() string {
	return string(c.component)
}

func (c *TrinoContainerBuilder) ContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			ContainerPort: 18080,
			Name:          "http",
			Protocol:      "TCP",
		},
	}
}

func (c *TrinoContainerBuilder) ResourceRequirements() corev1.ResourceRequirements {
	return *ConvertToResourceRequirements(c.resourceSpec)
}

func (c *TrinoContainerBuilder) VolumeMount() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      ConfigVolumeName(),
			MountPath: builder.ConfigDir,
		},
		{
			Name:      CatalogVolumeName(),
			MountPath: builder.ConfigDir + "/catalog",
		},
		{
			Name:      SchemaVolumeName(),
			MountPath: builder.LogDir + "/schemas",
		},
		{
			Name:      LogVolumeName(),
			MountPath: builder.LogDir,
		},
	}
}

func ConfigVolumeName() string {
	return "config"
}

func CatalogVolumeName() string {
	return "catalog"
}

func SchemaVolumeName() string {
	return "schema"
}

func LogVolumeName() string {
	return "log"
}
