package common

import (
	trinov1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const LogCfgName = "log.properties"

type RoleLoggingDataBuilder interface {
	MakeContainerLogData() map[string]string
}

type LoggingRecociler struct {
	GeneralResourceStyleReconciler[*trinov1alpha1.TrinoCluster, any]
	RoleLoggingDataBuilder RoleLoggingDataBuilder
	role                   Role
}

// NewLoggingReconciler new logging reconcile
func NewLoggingReconciler(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	mergedLabels map[string]string,
	mergedCfg any,
	logDataBuilder RoleLoggingDataBuilder,
	role Role,
) *LoggingRecociler {
	return &LoggingRecociler{
		GeneralResourceStyleReconciler: *NewGeneraResourceStyleReconciler[*trinov1alpha1.TrinoCluster, any](
			scheme,
			instance,
			client,
			groupName,
			mergedLabels,
			mergedCfg,
		),
		RoleLoggingDataBuilder: logDataBuilder,
		role:                   role,
	}
}

// Build log4j config map
func (l *LoggingRecociler) Build() (client.Object, error) {
	cmData := l.RoleLoggingDataBuilder.MakeContainerLogData()
	if len(cmData) == 0 {
		return nil, nil
	}
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CreateRoleGroupLoggingConfigMapName(l.Instance.Name, string(l.role), l.GroupName),
			Namespace: l.Instance.Namespace,
			Labels:    l.MergedLabels,
		},
		Data: cmData,
	}
	return obj, nil
}
