package worker

import (
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	Worker common.ContainerComponent = "worker"
)

func NewWorkerContainerBuilder(
	image string,
	imagePullPolicy corev1.PullPolicy,
	resourceSpec *trinov1alpha1.ResourcesSpec) *common.TrinoContainerBuilder {
	return common.NewTrinoContainerBuilder(image, imagePullPolicy, resourceSpec, Worker)
}

// workerWithVector coordinator with vector
func workerWithVector(
	containerLoggingSpec *trinov1alpha1.ContainerLoggingSpec,
	dep *appsv1.Deployment,
	vectorConfigMapName string) {
	common.WithVector([]string{string(Worker)}, containerLoggingSpec, dep, vectorConfigMapName)
}
