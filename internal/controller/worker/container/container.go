package container

import (
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
)

const (
	Worker common.ContainerComponent = "worker"
)

func NewWorkerContainerBuilder(image string,
	imagePullPolicy corev1.PullPolicy,
	resourceSpec *trinov1alpha1.ResourcesSpec) *common.TrinoContainerBuilder {
	return common.NewTrinoContainerBuilder(image, imagePullPolicy, resourceSpec, Worker)
}
