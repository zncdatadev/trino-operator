package common

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/zncdatadev/operator-go/pkg/builder"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/config/properties"
	"github.com/zncdatadev/operator-go/pkg/constants"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinosv1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const (
	JvmHeapFactor = 0.8
	DefaultJvmMem = "2G"
)

var (
	ServerTlsMountPath   = path.Join(constants.KubedoopTlsDir, "server")
	InternalTlsMountPath = path.Join(constants.KubedoopTlsDir, "internal")
	ClientTlsMountPath   = path.Join(constants.KubedoopTlsDir, "client")
)

func NewConfigReconciler(
	client *client.Client,
	coordiantorSvcFqdn string,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	info reconciler.RoleGroupInfo,
) reconciler.Reconciler {
	builder := NewConfigMapBuilder(
		client,
		info.GetFullName(),
		coordiantorSvcFqdn,
		clusterConfig,
		builder.Options{
			ClusterName:   info.GetClusterName(),
			RoleName:      info.GetRoleName(),
			RoleGroupName: info.GetGroupName(),
			Labels:        info.GetLabels(),
			Annotations:   info.GetAnnotations(),
		},
	)

	return reconciler.NewGenericResourceReconciler(
		client,
		info.GetFullName(),
		builder,
	)
}

var _ builder.ConfigBuilder = &ConfigMapBuilder{}

type ConfigMapBuilder struct {
	builder.ConfigMapBuilder

	TrinoConfig        *trinosv1alpha1.ConfigSpec
	ClusterConfig      *trinosv1alpha1.ClusterConfigSpec
	RoleName           string
	CoordiantorSvcFqdn string
	ClusterName        string
}

func NewConfigMapBuilder(
	client *client.Client,
	name string,
	coordinatorSvcFqdn string,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	options builder.Options,
) *ConfigMapBuilder {
	return &ConfigMapBuilder{
		ConfigMapBuilder: *builder.NewConfigMapBuilder(
			client,
			name,
			options.Labels,
			options.Annotations,
		),
		RoleName:           options.RoleName,
		ClusterName:        options.ClusterName,
		CoordiantorSvcFqdn: coordinatorSvcFqdn,
		ClusterConfig:      clusterConfig,
	}
}

func (b *ConfigMapBuilder) Build(ctx context.Context) (ctrlclient.Object, error) {
	properties := map[string]*properties.Properties{
		"config.properties": b.getConfigProperties(),
		"node.properties":   b.getNodeProperties(),
		// "log.properties":    b.getLogProperties(),
		// "security.properties": b.getSecurityProperties(),
	}

	for k, v := range properties {
		value, err := v.Marshal()
		if err != nil {
			return nil, err
		}
		b.AddItem(k, value)
	}

	b.AddItem("jvm.config", b.getJvmProperties())

	return b.GetObject(), nil
}

func (b *ConfigMapBuilder) getDiscoveryUri(port int) string {
	schema := "http"
	return schema + "://" + b.CoordiantorSvcFqdn + ":" + strconv.Itoa(port)
}

func (b *ConfigMapBuilder) getConfigProperties() *properties.Properties {
	port := int(HttpPort)
	p := properties.NewProperties()

	if b.RoleName == "coordinator" {
		p.Add("coordinator", "true")
	} else {
		p.Add("coordinator", "false")
	}

	p.Add("http-server.http.port", strconv.Itoa(port))
	p.Add("query.max-memory", "5GB")
	p.Add("query.max-memory-per-node", "1GB")
	p.Add("discovery.uri", b.getDiscoveryUri(port))
	p.Add("http-server.log.enabled", "false")

	return p
}

func (b *ConfigMapBuilder) getNodeProperties() *properties.Properties {
	p := properties.NewProperties()

	p.Add("node.environment", strings.ReplaceAll(b.ClusterName, "-", "_"))
	return p
}

// func (b *ConfigMapBuilder) getLogProperties() *properties.Properties {
// 	panic("implement me")
// }

// func (b *ConfigMapBuilder) getSecurityProperties() *properties.Properties {
// 	panic("implement me")
// }

// Only support K, M, G.
func (b *ConfigMapBuilder) getHeapSize(factor float64) string {
	memory := resource.MustParse(DefaultJvmMem)

	if b.TrinoConfig != nil && b.TrinoConfig.Resources != nil && b.TrinoConfig.Resources.Memory != nil {
		memory = b.TrinoConfig.Resources.Memory.Limit
	}
	value := memory.Value()

	heapSize := memory.ToDec().AsApproximateFloat64()

	var unit string
	switch {
	case value >= 1<<40:
		unit = "G"
		heapSize = heapSize / (1 << 30)
	case value >= 1<<30:
		unit = "M"
		heapSize = heapSize / (1 << 20)
	case value >= 1<<20:
		unit = "K"
		heapSize = heapSize / (1 << 10)
	default:
		panic("invalid memory size, must be greater than 1K. current value: " + memory.String())
	}
	return fmt.Sprintf("%d%s", int(heapSize*factor), unit)
}

func (b *ConfigMapBuilder) getJvmProperties() string {

	jvm := `
-server
-Xmx` + b.getHeapSize(JvmHeapFactor) + `
-Xms` + b.getHeapSize(JvmHeapFactor) + `
-XX:InitialRAMPercentage=80
-XX:MaxRAMPercentage=80
-XX:G1HeapRegionSize=32M
-XX:+ExplicitGCInvokesConcurrent
-XX:+ExitOnOutOfMemoryError
-XX:+HeapDumpOnOutOfMemoryError
-XX:-OmitStackTraceInFastThrow
-XX:ReservedCodeCacheSize=512M
-XX:PerMethodRecompilationCutoff=10000
-XX:PerBytecodeRecompilationCutoff=10000
-Djdk.attach.allowAttachSelf=true
-Djdk.nio.maxCachedBufferSize=2000000
-Dfile.encoding=UTF-8
# Allow loading dynamic agent used by JOL
-XX:+EnableDynamicAgentLoading
-XX:+UnlockDiagnosticVMOptions
-XX:G1NumCollectionsKeepPinned=10000000
`

	return util.IndentTab4Spaces(jvm)
}
