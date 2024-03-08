package coordinator

import (
	"context"

	trinov1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	"github.com/zncdata-labs/trino-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceReconciler struct {
	common.GeneralResourceStyleReconciler[*trinov1alpha1.TrinoCluster, *trinov1alpha1.RoleGroupSpec]
}

// NewService new a ServiceReconcile
func NewService(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	mergedLabels map[string]string,
	mergedCfg *trinov1alpha1.RoleGroupSpec,

) *ServiceReconciler {
	return &ServiceReconciler{
		GeneralResourceStyleReconciler: *common.NewGeneraResourceStyleReconciler[*trinov1alpha1.TrinoCluster,
			*trinov1alpha1.RoleGroupSpec](
			scheme,
			instance,
			client,
			groupName,
			mergedLabels,
			mergedCfg,
		),
	}
}

// Build implements the ResourceBuilder interface
func (s *ServiceReconciler) Build(_ context.Context) (client.Object, error) {
	instance := s.Instance
	roleGroupName := s.GroupName
	svcSpec := s.getServiceSpec()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        common.CreateServiceName(instance.Name, string(common.Coordinator), roleGroupName),
			Namespace:   instance.Namespace,
			Labels:      s.MergedLabels,
			Annotations: svcSpec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       svcSpec.Port,
					Name:       "http",
					Protocol:   "TCP",
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: s.MergedLabels,
			Type:     svcSpec.Type,
		},
	}
	return svc, nil
}

// get service spec
func (s *ServiceReconciler) getServiceSpec() *trinov1alpha1.ServiceSpec {
	return getServiceSpec(s.Instance)
}
