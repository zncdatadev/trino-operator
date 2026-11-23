package controller

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/constant"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"k8s.io/apimachinery/pkg/api/resource"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const (
	configFileName         = "config.properties"
	nodeFileName           = "node.properties"
	secretFileName         = "secret.properties"
	jvmFileName            = "jvm.config"
	logFileName            = "log.properties"
	defaultQueryMaxMemory  = "5GB"
	defaultQueryMaxPerNode = "1000MB"
	defaultJVMMemory       = "2G"
	defaultStorageCapacity = "1Gi"
	jvmMaxHeapOption       = "-Xmx"
	jvmHeapFactor          = 0.8
	httpScheme             = "http"
	httpsScheme            = "https"
	stringFalse            = "false"
	stringTrue             = "true"
)

func trinoConfigFiles(
	cr *trinov1alpha1.TrinoCluster,
	buildCtx *reconciler.RoleGroupBuildContext,
	authProperties map[string]string,
) map[string]map[string]string {
	configProperties := map[string]string{
		"coordinator":                        strconv.FormatBool(buildCtx.RoleName == trinov1alpha1.TrinoCoordinatorRoleName),
		"node-scheduler.include-coordinator": stringFalse,
		"node.internal-address-source":       "FQDN",
		"http-server.log.enabled":            stringFalse,
		"discovery.uri":                      discoveryURI(cr),
		"log.compression":                    "none",
		"log.format":                         "json",
		"log.max-size":                       "5MB",
		"log.max-total-size":                 "10MB",
		"log.path": path.Join(
			constant.KubedoopLogDir,
			"trino",
			buildCtx.RoleName+".airlift.json",
		),
	}

	queryMaxMemory, queryMaxMemoryPerNode, includePerNode := resolveQueryMemory(
		cr,
		buildCtx.RoleName,
		buildCtx.RoleGroupName,
	)
	configProperties["query.max-memory"] = queryMaxMemory
	if includePerNode {
		configProperties["query.max-memory-per-node"] = queryMaxMemoryPerNode
	}

	if tlsEnabled(cr) {
		configProperties["internal-communication.https.required"] = stringTrue
		configProperties["internal-communication.shared-secret"] = "${ENV:" + internalSharedSecretEnvName + "}"
		configProperties["http-server.https.enabled"] = stringTrue
		configProperties["http-server.https.port"] = strconv.Itoa(int(trinov1alpha1.HttpsPort))
		configProperties["http-server.https.keystore.path"] = path.Join(serverTLSMountPath, "keystore.p12")
		configProperties["http-server.https.keystore.key"] = defaultTLSPassphrase
		configProperties["http-server.https.truststore.path"] = path.Join(serverTLSMountPath, "truststore.p12")
		configProperties["http-server.https.truststore.key"] = defaultTLSPassphrase
		configProperties["internal-communication.https.keystore.path"] = path.Join(internalTLSMountPath, "keystore.p12")
		configProperties["internal-communication.https.keystore.key"] = defaultTLSPassphrase
		configProperties["internal-communication.https.truststore.path"] = path.Join(internalTLSMountPath, "truststore.p12")
		configProperties["internal-communication.https.truststore.key"] = defaultTLSPassphrase
	} else {
		configProperties["http-server.http.port"] = strconv.Itoa(int(trinov1alpha1.HttpPort))
	}

	if buildCtx.RoleName == trinov1alpha1.TrinoCoordinatorRoleName {
		for key, value := range authProperties {
			configProperties[key] = value
		}
	}

	files := map[string]map[string]string{
		configFileName: configProperties,
		jvmFileName:    jvmConfig(buildCtx.EffectiveConfig()),
		logFileName: {
			"": "info",
		},
		nodeFileName: {
			"node.environment": strings.ReplaceAll(cr.Name, "-", "_"),
		},
		secretFileName: {
			"networkaddress.cache.negative.ttl": "0",
			"networkaddress.cache.ttl":          "30",
		},
	}

	if cr.Spec.ClusterConfig != nil {
		for name, properties := range cr.Spec.ClusterConfig.CatalogProperties {
			fileName := fmt.Sprintf("catalog-%s.properties", name)
			files[fileName] = cloneStringMap(properties)
		}
	}

	return files
}

func discoveryURI(cr *trinov1alpha1.TrinoCluster) string {
	scheme := httpScheme
	port := trinov1alpha1.HttpPort
	if tlsEnabled(cr) {
		scheme = httpsScheme
		port = trinov1alpha1.HttpsPort
	}

	serviceName := cr.Name + "-" + trinov1alpha1.TrinoCoordinatorRoleName
	host := serviceName + "." + cr.Namespace + ".svc.cluster.local"
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// resolveQueryMemory keeps the legacy no-config behavior while applying defaults at consumption
// time. This avoids CRD leaf defaults shadowing values inherited from the role.
func resolveQueryMemory(
	cr *trinov1alpha1.TrinoCluster,
	roleName string,
	roleGroupName string,
) (queryMaxMemory string, queryMaxMemoryPerNode string, includePerNode bool) {
	queryMaxMemory = trinov1alpha1.DefaultQueryMaxMemory
	queryMaxMemoryPerNode = defaultQueryMaxPerNode

	roleConfig, roleGroupConfig := roleAndGroupConfig(cr, roleName, roleGroupName)
	if roleConfig != nil || roleGroupConfig != nil {
		queryMaxMemory = defaultQueryMaxMemory
		includePerNode = true
	}
	if roleConfig != nil {
		if roleConfig.QueryMaxMemory != "" {
			queryMaxMemory = roleConfig.QueryMaxMemory
		}
		if roleConfig.QueryMaxMemoryPerNode != "" {
			queryMaxMemoryPerNode = roleConfig.QueryMaxMemoryPerNode
			includePerNode = true
		}
	}
	if roleGroupConfig != nil {
		if roleGroupConfig.QueryMaxMemory != "" {
			queryMaxMemory = roleGroupConfig.QueryMaxMemory
		}
		if roleGroupConfig.QueryMaxMemoryPerNode != "" {
			queryMaxMemoryPerNode = roleGroupConfig.QueryMaxMemoryPerNode
			includePerNode = true
		}
	}
	return queryMaxMemory, queryMaxMemoryPerNode, includePerNode
}

func roleAndGroupConfig(
	cr *trinov1alpha1.TrinoCluster,
	roleName string,
	roleGroupName string,
) (*trinov1alpha1.ConfigSpec, *trinov1alpha1.ConfigSpec) {
	var roleConfig *trinov1alpha1.ConfigSpec
	var groups map[string]*trinov1alpha1.RoleGroupSpec
	switch roleName {
	case trinov1alpha1.TrinoCoordinatorRoleName:
		if cr.Spec.Coordinators != nil {
			roleConfig = cr.Spec.Coordinators.Config
			groups = cr.Spec.Coordinators.RoleGroups
		}
	case trinov1alpha1.TrinoWorkerRoleName:
		if cr.Spec.Workers != nil {
			roleConfig = cr.Spec.Workers.Config
			groups = cr.Spec.Workers.RoleGroups
		}
	}
	if group := groups[roleGroupName]; group != nil {
		return roleConfig, group.Config
	}
	return roleConfig, nil
}

func jvmConfig(config *commonsv1alpha1.RoleGroupConfigSpec) map[string]string {
	memory := resource.MustParse(defaultJVMMemory)
	if config != nil && config.Resources != nil && config.Resources.Memory != nil &&
		config.Resources.Memory.Limit != nil {
		memory = config.Resources.Memory.Limit.DeepCopy()
	}

	heapSize := memory.ToDec().AsApproximateFloat64()
	value := memory.Value()
	unit := "K"
	switch {
	case value >= 1<<40:
		unit = "G"
		heapSize /= 1 << 30
	case value >= 1<<30:
		unit = "M"
		heapSize /= 1 << 20
	case value >= 1<<20:
		heapSize /= 1 << 10
	default:
		heapSize /= 1 << 10
	}
	heap := fmt.Sprintf("%d%s", int(heapSize*jvmHeapFactor), unit)

	return map[string]string{
		"-server":                            "",
		jvmMaxHeapOption:                     heap,
		"-Xms":                               heap,
		"-XX:InitialRAMPercentage":           "80",
		"-XX:MaxRAMPercentage":               "80",
		"-XX:G1HeapRegionSize":               "32M",
		"-XX:+ExplicitGCInvokesConcurrent":   "",
		"-XX:+ExitOnOutOfMemoryError":        "",
		"-XX:+HeapDumpOnOutOfMemoryError":    "",
		"-XX:-OmitStackTraceInFastThrow":     "",
		"-XX:ReservedCodeCacheSize":          "512M",
		"-XX:PerMethodRecompilationCutoff":   "10000",
		"-XX:PerBytecodeRecompilationCutoff": "10000",
		"-Djdk.attach.allowAttachSelf":       stringTrue,
		"-Djdk.nio.maxCachedBufferSize":      "2000000",
		"-Dfile.encoding":                    "UTF-8",
		"-XX:+EnableDynamicAgentLoading":     "",
		"-XX:+UnlockDiagnosticVMOptions":     "",
		"-XX:G1NumCollectionsKeepPinned":     "10000000",
		"-Djava.net.ssl.trustStore":          path.Join(clientTLSMountPath, "truststore.p12"),
		"-Djava.net.ssl.trustStorePassword":  defaultTLSPassphrase,
		"-Djava.net.ssl.trustStoreType":      "PKCS12",
		"-Djava.secret.properties":           path.Join(constant.KubedoopConfigDir, secretFileName),
		"-javaagent:": fmt.Sprintf(
			"%s=%d:%s",
			path.Join(constant.KubedoopJmxDir, "jmx_prometheus_javaagent.jar"),
			trinov1alpha1.MetricsPort,
			path.Join(constant.KubedoopJmxDir, "config.yaml"),
		),
	}
}

// jvmConfigAdapter renders the map-shaped configOverrides API into Trino's
// one-option-per-line jvm.config format. Empty values represent flag options;
// -X size options and keys ending in ':' or '=' concatenate their value.
type jvmConfigAdapter struct{}

func (jvmConfigAdapter) Marshal(data map[string]string) (string, error) {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := data[key]
		if strings.ContainsAny(key, "\r\n") || strings.ContainsAny(value, "\r\n") {
			return "", fmt.Errorf("JVM option %q contains a newline", key)
		}

		line := key
		if value != "" {
			switch {
			case key == jvmMaxHeapOption, key == "-Xms", key == "-Xss",
				strings.HasSuffix(key, ":"), strings.HasSuffix(key, "="):
				line += value
			default:
				line += "=" + value
			}
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return "", nil
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
