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
	"github.com/zncdatadev/operator-go/pkg/productlogging"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinosv1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/controller/common/authz"
)

const (
	JvmHeapFactor = 0.8
	DefaultJvmMem = "2G"
)

var (
	ServerTlsMountPath   = path.Join(constants.KubedoopTlsDir, "server")
	InternalTlsMountPath = path.Join(constants.KubedoopTlsDir, "internal")
	ClientTlsPath        = path.Join(constants.KubedoopTlsDir, "client")
	DefaultTlsPassphrase = "changeit"
)

func NewConfigReconciler(
	client *client.Client,
	coordiantorSvcFqdn string,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	trinoConfig *trinosv1alpha1.ConfigSpec,
	info reconciler.RoleGroupInfo,
) reconciler.Reconciler {
	builder := NewConfigMapBuilder(
		client,
		info.GetFullName(),
		coordiantorSvcFqdn,
		clusterConfig,
		trinoConfig,
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

	TrinoConfig   *trinosv1alpha1.ConfigSpec
	ClusterConfig *trinosv1alpha1.ClusterConfigSpec
	TrinoConig    *trinosv1alpha1.ConfigSpec

	CoordiantorSvcFqdn string

	ClusterName   string
	RoleName      string
	RoleGroupName string
}

func NewConfigMapBuilder(
	client *client.Client,
	name string,
	coordinatorSvcFqdn string,
	clusterConfig *trinosv1alpha1.ClusterConfigSpec,
	trinoConfig *trinosv1alpha1.ConfigSpec,
	options builder.Options,
) *ConfigMapBuilder {
	return &ConfigMapBuilder{
		ConfigMapBuilder: *builder.NewConfigMapBuilder(
			client,
			name,
			options.Labels,
			options.Annotations,
		),
		CoordiantorSvcFqdn: coordinatorSvcFqdn,
		ClusterConfig:      clusterConfig,
		TrinoConfig:        trinoConfig,

		ClusterName:   options.ClusterName,
		RoleName:      options.RoleName,
		RoleGroupName: options.RoleGroupName,
	}
}

func (b *ConfigMapBuilder) Build(ctx context.Context) (ctrlclient.Object, error) {
	configProperties, err := b.getConfigProperties(ctx)
	if err != nil {
		return nil, err
	}
	s, err := configProperties.Marshal()
	if err != nil {
		return nil, err
	}
	b.AddItem("config.properties", s)

	nodeProperties := b.getNodeProperties()
	s, err = nodeProperties.Marshal()
	if err != nil {
		return nil, err
	}

	secretProperties := b.getSecurityProperties()
	s, err = secretProperties.Marshal()
	if err != nil {
		return nil, err
	}

	b.AddItem("jvm.config", b.getJvmProperties())
	b.AddItem("log.properties", `=info
`)

	if b.ClusterConfig.VectorAggregatorConfigMapName != "" {
		s, err := productlogging.MakeVectorYaml(
			ctx, b.Client.Client,
			b.Client.GetOwnerNamespace(),
			b.ClusterName,
			b.RoleName,
			b.RoleGroupName,
			b.ClusterConfig.VectorAggregatorConfigMapName,
		)
		if err != nil {
			return nil, err
		}
		b.AddItem(builder.VectorConfigFile, s)
	}

	return b.GetObject(), nil
}

func (b *ConfigMapBuilder) getDiscoveryUri() string {
	schema := "http"
	port := int(trinosv1alpha1.HttpPort)
	if b.enabledTls() {
		schema = "https"
		port = int(trinosv1alpha1.HttpsPort)
	}
	return schema + "://" + b.CoordiantorSvcFqdn + ":" + strconv.Itoa(port)
}

func (b *ConfigMapBuilder) enabledTls() bool {
	return b.ClusterConfig != nil && b.ClusterConfig.Tls != nil
}

func (b *ConfigMapBuilder) getConfigProperties(ctx context.Context) (*properties.Properties, error) {
	p := properties.NewProperties()

	if b.RoleName == "coordinator" {
		p.Add("coordinator", "true")
	} else {
		p.Add("coordinator", "false")
	}
	p.Add("node-scheduler.include-coordinator", "false")

	if b.TrinoConfig != nil {
		p.Add("query.max-memory", b.TrinoConfig.QueryMaxMemory)
		p.Add("query.max-memory-per-node", b.TrinoConfig.QueryMaxMemoryPerNode)
	} else {
		p.Add("query.max-memory", trinosv1alpha1.DefaultQueryMaxMemory)
	}

	p.Add("node.internal-address-source", "FQDN")
	p.Add("http-server.log.enabled", "false")
	p.Add("discovery.uri", b.getDiscoveryUri())
	if b.enabledTls() {
		p.Add("internal-communication.https.required", "true")

		p.Add("internal-communication.shared-secret", fmt.Sprintf("${ENV:%s}", InternalSharedSecretEnvName))
		p.Add("http-server.https.enabled", "true")
		p.Add("http-server.https.port", strconv.Itoa(int(trinosv1alpha1.HttpsPort)))

		p.Add("http-server.https.keystore.path", path.Join(ServerTlsMountPath, "keystore.p12"))
		p.Add("http-server.https.keystore.key", DefaultTlsPassphrase)
		p.Add("http-server.https.truststore.path", path.Join(ServerTlsMountPath, "truststore.p12"))
		p.Add("http-server.https.truststore.key", DefaultTlsPassphrase)

		p.Add("internal-communication.https.keystore.path", path.Join(ServerTlsMountPath, "keystore.p12"))
		p.Add("internal-communication.https.keystore.key", DefaultTlsPassphrase)
		p.Add("internal-communication.https.truststore.path", path.Join(ServerTlsMountPath, "truststore.p12"))
		p.Add("internal-communication.https.truststore.key", DefaultTlsPassphrase)
	} else {
		p.Add("http-server.http.port", strconv.Itoa(int(trinosv1alpha1.HttpPort)))
	}

	p.Add("log.compression", "none")
	p.Add("log.format", "json")
	p.Add("log.max-size", "5MB")
	p.Add("log.max-total-size", "10MB")
	p.Add("log.path", path.Join(constants.KubedoopLogDir, "trino", "airlift.json"))

	if b.ClusterConfig.Authentication != nil {
		authentication, err := authz.NewAuthentication(ctx, b.Client, b.ClusterConfig.Authentication)
		if err != nil {
			return nil, err
		}
		for _, key := range authentication.GetConfigProperties().Keys() {
			value, _ := authentication.GetConfigProperties().Get(key)
			p.Add(key, value)
		}
	}

	return p, nil
}

func (b *ConfigMapBuilder) getNodeProperties() *properties.Properties {
	p := properties.NewProperties()

	p.Add("node.environment", strings.ReplaceAll(b.ClusterName, "-", "_"))
	return p
}

func (b *ConfigMapBuilder) getSecurityProperties() *properties.Properties {
	p := properties.NewProperties()
	p.Add("networkaddress.cache.negative.ttl", "0")
	p.Add("networkaddress.cache.ttl", "30")
	return p
}

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

	jvm := `-server
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
-Djava.net.ssl.trustStore=` + path.Join(constants.KubedoopTlsDir, "client", "truststore.p12") + `
-Djava.net.ssl.trustStorePassword=` + DefaultTlsPassphrase + `
-Djava.net.ssl.trustStoreType=PKCS12
-Djava.secret.properties=` + path.Join(constants.KubedoopConfigDir, "secret.properties") + `
-javaagent:` + path.Join(constants.KubedoopJmxDir, "jmx_prometheus_javaagent.jar") + `=9404:` + path.Join(constants.KubedoopJmxDir, "config.yaml") + `
`
	return util.IndentTab4Spaces(jvm)
}
