package coordinator

import (
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	logger = ctrl.Log.WithName("coordinator")
)

func createIngName(instanceName string, groupName string) string {
	return common.NewResourceNameGenerator(instanceName, "", groupName).GenerateResourceName("")
}
func createCoordinatorConfigmapName(instanceName string, groupName string) string {
	return common.NewResourceNameGenerator(instanceName, string(common.Coordinator), groupName).GenerateResourceName("")
}

func createCoordinatorDeploymentName(instanceName string, groupName string) string {
	return common.NewResourceNameGenerator(instanceName, string(common.Coordinator), groupName).GenerateResourceName("")
}
func getServiceSpec(instance *trinov1alpha1.TrinoCluster) *trinov1alpha1.ServiceSpec {
	spec := instance.Spec.ClusterConfig.Service
	if spec == nil {
		spec = &trinov1alpha1.ServiceSpec{
			Type: trinov1alpha1.ServiceType,
			Port: trinov1alpha1.ServicePort,
		}
	}
	return spec
}

func GetRole() common.Role {
	return common.Coordinator
}
