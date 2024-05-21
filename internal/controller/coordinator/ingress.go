package coordinator

import (
	"context"
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressReconciler struct {
	common.GeneralResourceStyleReconciler[*trinov1alpha1.TrinoCluster, *trinov1alpha1.RoleGroupSpec]
}

func NewIngress(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	mergedLabels map[string]string,
	mergedCfg *trinov1alpha1.RoleGroupSpec,
) *IngressReconciler {
	return &IngressReconciler{
		GeneralResourceStyleReconciler: *common.NewGeneraResourceStyleReconciler[*trinov1alpha1.TrinoCluster,
			*trinov1alpha1.RoleGroupSpec](
			scheme,
			instance,
			client,
			groupName,
			mergedLabels,
			mergedCfg),
	}
}

// Build implements the ResourceBuilder interface
func (i *IngressReconciler) Build(_ context.Context) (client.Object, error) {
	ingressSpec := i.getIngressSpec()
	pt := v1.PathTypePrefix
	ing := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      createIngName(i.Instance.Name, i.GroupName),
			Namespace: i.Instance.Namespace,
			Labels:    i.MergedLabels,
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: ingressSpec.Host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pt,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: common.CreateServiceName(i.Instance.Name, string(common.Coordinator),
												i.GroupName),
											Port: v1.ServiceBackendPort{
												Number: i.getServicePort(),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return ing, nil
}

// get ingress spec
func (i *IngressReconciler) getIngressSpec() *trinov1alpha1.IngressSpec {
	spec := i.Instance.Spec.ClusterConfig.Ingress
	if spec == nil {
		spec = &trinov1alpha1.IngressSpec{
			Host:    "trino.example.com",
			Enabled: true,
		}
	}
	return spec
}

// get service port
func (i *IngressReconciler) getServicePort() int32 {
	svcSpec := getServiceSpec(i.Instance)
	return svcSpec.Port
}
