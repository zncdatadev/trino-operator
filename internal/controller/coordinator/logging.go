package coordinator

import (
	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/common"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCoordinatorLogging(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	mergedLabels map[string]string,
	mergedCfg any,
	logDataBuilder common.RoleLoggingDataBuilder,
	role common.Role,
) *common.LoggingRecociler {
	return common.NewLoggingReconciler(scheme, instance, client, groupName, mergedLabels, mergedCfg, logDataBuilder, role)
}

type LogDataBuilder struct {
	cfg *trinov1alpha1.RoleGroupSpec
}

// MakeContainerLogData MakeContainerLog4jData implement RoleLoggingDataBuilder
func (c *LogDataBuilder) MakeContainerLogData() map[string]string {
	cfg := c.cfg
	data := make(map[string]string)
	// logger data
	if logging := cfg.Config.Logging; logging != nil {
		if container := logging.Trino; container != nil {
			loggers := container.Loggers
			if len(loggers) > 0 {
				var lines string
				for logger, level := range loggers {
					lines = lines + logger + "=" + level.Level + "\n"
				}
				data[common.LogCfgName] = lines
			}
		}

	}
	return data
}
