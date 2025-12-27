package common

import (
	"fmt"
	"strconv"

	"github.com/zncdatadev/operator-go/pkg/builder"
	"github.com/zncdatadev/operator-go/pkg/client"
	opconstants "github.com/zncdatadev/operator-go/pkg/constants"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	trinosv1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// NewRoleGroupMetricsService creates a metrics service reconciler using a simple function approach
// This creates a headless service for metrics with Prometheus labels and annotations
func NewRoleGroupMetricsService(
	client *client.Client,
	roleGroupInfo *reconciler.RoleGroupInfo,
) reconciler.Reconciler {
	metricsPort := trinosv1alpha1.MetricsPort

	// Create service ports
	servicePorts := []corev1.ContainerPort{
		{
			Name:          trinosv1alpha1.MetricsPortName,
			ContainerPort: metricsPort,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	// Create service name with -metrics suffix
	serviceName := CreateServiceMetricsName(roleGroupInfo)

	// Prepare labels (copy from roleGroupInfo and add metrics labels)
	labels := make(map[string]string)
	for k, v := range roleGroupInfo.GetLabels() {
		labels[k] = v
	}
	labels["prometheus.io/scrape"] = "true"

	// Prepare annotations (copy from roleGroupInfo and add Prometheus annotations)
	annotations := make(map[string]string)
	for k, v := range roleGroupInfo.GetAnnotations() {
		annotations[k] = v
	}
	annotations["prometheus.io/scrape"] = "true"
	// annotations["prometheus.io/path"] = "metrics" // Default path is /metrics, so no need to set it explicitly
	annotations["prometheus.io/port"] = strconv.Itoa(int(metricsPort))
	annotations["prometheus.io/scheme"] = HttpScheme

	// Create base service builder
	baseBuilder := builder.NewServiceBuilder(
		client,
		serviceName,
		servicePorts,
		func(sbo *builder.ServiceBuilderOptions) {
			sbo.Headless = true
			sbo.ListenerClass = opconstants.ClusterInternal
			sbo.Labels = labels
			sbo.MatchingLabels = roleGroupInfo.GetLabels() // Use original labels for matching
			sbo.Annotations = annotations
		},
	)

	return reconciler.NewGenericResourceReconciler(
		client,
		baseBuilder,
	)
}

func CreateServiceMetricsName(roleGroupInfo *reconciler.RoleGroupInfo) string {
	return fmt.Sprintf("%s-metrics", roleGroupInfo.GetFullName())
}
