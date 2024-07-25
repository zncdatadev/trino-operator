package common

import (
	"github.com/zncdatadev/operator-go/pkg/builder"
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

const VectorConfigVolumeName = "config"
const VectorLogVolumeName = "log"

func IsVectorEnabled(loggingSpec *trinov1alpha1.ContainerLoggingSpec) bool {
	return loggingSpec != nil && loggingSpec.EnableVectorAgent
}

// WithVector coordinator with vector
func WithVector(
	logProvider []string,
	containerLoggingSpec *trinov1alpha1.ContainerLoggingSpec,
	dep *appsv1.Deployment,
	vectorConfigMapName string) {
	if !IsVectorEnabled(containerLoggingSpec) {
		return
	}
	decorator := builder.VectorDecorator{
		WorkloadObject:           dep,
		LogVolumeName:            VectorLogVolumeName,
		VectorConfigVolumeName:   VectorConfigVolumeName,
		VectorConfigMapName:      vectorConfigMapName,
		LogProviderContainerName: logProvider,
	}

	err := decorator.Decorate()
	if err != nil {
		return
	}
}
