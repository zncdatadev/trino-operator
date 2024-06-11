package common

import (
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/util"
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
	tmpl := `
      prepare_signal_handlers()
      {
          unset term_child_pid
          unset term_kill_needed
          trap 'handle_term_signal' TERM
      }

      handle_term_signal()
      {
          if [ "${term_child_pid}" ]; then
              kill -TERM "${term_child_pid}" 2>/dev/null
          else
              term_kill_needed="yes"
          fi
      }

      wait_for_termination()
      {
          set +e
          term_child_pid=$1
          if [[ -v term_kill_needed ]]; then
              kill -TERM "${term_child_pid}" 2>/dev/null
          fi
          wait ${term_child_pid} 2>/dev/null
          trap - TERM
          wait ${term_child_pid} 2>/dev/null
          set -e
      }

      rm -f {{ .LogDir }}/_vector/shutdown
      prepare_signal_handlers

      set -xeuo pipefail
      launcher_opts=(--etc-dir /etc/trino)
      if ! grep -s -q 'node.id' /etc/trino/node.properties; then
        launcher_opts+=("-Dnode.id=${HOSTNAME}")
      fi
      exec /usr/lib/trino/bin/launcher run "${launcher_opts[@]}" "$@"

      wait_for_termination $!
      mkdir -p {{ .LogDir }}/_vector && touch {{ .LogDir }}/_vector/shutdown
`
	data := map[string]interface{}{"LogDir": LogDir}
	parser := util.TemplateParser{
		Value:    data,
		Template: tmpl,
	}
	if res, err := parser.Parse(); err == nil {
		return []string{res}
	} else {
		panic(err)
	}
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
			MountPath: "/etc/trino",
		},
		{
			Name:      CatalogVolumeName(),
			MountPath: "/etc/trino/catalog",
		},
		{
			Name:      SchemaVolumeName(),
			MountPath: "/etc/trino/schemas",
		},
		{
			Name:      LogVolumeName(),
			MountPath: LogDir,
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
