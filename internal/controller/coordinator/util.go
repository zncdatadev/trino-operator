package coordinator

import (
	trinov1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	"github.com/zncdata-labs/trino-operator/internal/common"
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
			Type: "ClusterIP",
			Port: 9083,
		}
	}
	return spec
}
