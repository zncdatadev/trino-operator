package common

import (
	"context"
	"github.com/zncdatadev/trino-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const LogDir = "/zncdata/log"
const Vector ContainerComponent = "vector"
const VectorConfigVolumeName = "config"
const VectorLogVolumeName = "log"

func NewVectorContainerBuilder() *VectorContainerBuilder {
	return &VectorContainerBuilder{
		ContainerBuilder: ContainerBuilder{
			Image:           "timberio/vector:0.38.0-alpine",
			ImagePullPolicy: "Always",
		},
	}
}

type VectorContainerTypes interface {
	ContainerName
	VolumeMount
	CommandArgs
	Command
}

var _ VectorContainerTypes = VectorContainerBuilder{}

type VectorContainerBuilder struct {
	ContainerBuilder
}

func (v VectorContainerBuilder) ContainerName() string {
	return string(Vector)
}

func (v VectorContainerBuilder) VolumeMount() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      VectorConfigVolumeName,
			MountPath: "/zncdata/config",
		},
		{
			Name:      VectorLogVolumeName,
			MountPath: LogDir,
		},
	}
}

func (v VectorContainerBuilder) CommandArgs() []string {
	return []string{
		`log_dir="/zncdata/log/_vector"

vector --config /zncdata/config/vector.yaml &
vector_pid=$!

if [ ! -f "$log_dir/shutdown" ]; then
  mkdir -p "$log_dir"
fi

previous_count=$(ls -1 "$log_dir" | wc -l)

while true; do
    current_count=$(ls -1 "$log_dir" | wc -l)

    if [ "$current_count" -gt "$previous_count" ]; then
        new_file=$(ls -1 "$log_dir" | tail -n 1)
        echo "New file created: $new_file"

        previous_count=$current_count
    fi

    if [ -f "$log_dir/shutdown" ]; then
        kill $vector_pid
        break
    fi

    sleep 1
done
`,
	}
}

func (v VectorContainerBuilder) Command() []string {
	return []string{
		"ash",
		"-x",
		"-euo",
		"pipefail",
		"-c",
	}
}

func MakeVectorYaml(
	ctx context.Context,
	client client.Client,
	namespace string,
	cluster string,
	role Role,
	groupName string,
	vectorAggregatorDiscovery string) *string {
	data := map[string]interface{}{
		"LogDir":                  LogDir,
		"Namespace":               namespace,
		"Cluster":                 cluster,
		"Role":                    string(role),
		"GroupName":               groupName,
		"VectorAggregatorAddress": vectorAggregatorDiscoveryURI(ctx, client, namespace, vectorAggregatorDiscovery),
	}
	var tmpl = `
api:
  enabled: true
data_dir: /zncdata/vector/var
log_schema:
  host_key: "pod"
sources:
  files_airlift:
    type: "file"
    include:
      - "{{.LogDir}}/*/*.airlift.json"

transforms:
  processed_files_airlift:
    inputs:
      - files_airlift
    type: remap
    source: |
      parsed_event = parse_json!(string!(.message))
      .message = join!(compact([parsed_event.message, parsed_event.stackTrace]), "\n")
      .timestamp = parse_timestamp!(parsed_event.timestamp, "%Y-%m-%dT%H:%M:%S.%fZ")
      .logger = parsed_event.logger
      .level = parsed_event.level
      .thread = parsed_event.thread
  extended_logs_files:
    inputs:
      - processed_files_*
    type: remap
    source: |
      . |= parse_regex!(.file, r'^/stackable/log/(?P<container>.*?)/(?P<file>.*?)$')
      del(.source_type)
  extended_logs:
    inputs:
      - extended_logs_*
    type: remap
    source: |
      .namespace = {{.Namespace}}
      .cluster = {{.Cluster}}
      .role = {{.Role}}
      .roleGroup = {{.GroupName}}
sinks:
  aggregator:
    inputs:
      - extended_logs
    type: vector
    address: {{.VectorAggregatorAddress}}
`
	parser := util.TemplateParser{
		Value:    data,
		Template: tmpl,
	}

	str, err := parser.Parse()
	if err != nil {
		panic(err)
	}
	return &str
}

func vectorAggregatorDiscoveryURI(
	ctx context.Context,
	client client.Client,
	namespace string,
	discoveryConfigName string) *string {
	if discoveryConfigName != "" {
		cli := ResourceClient{
			Ctx:       ctx,
			Client:    client,
			Namespace: namespace,
		}
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      discoveryConfigName,
				Namespace: namespace,
			},
		}
		err := cli.Get(cm)
		if err != nil {
			return nil
		}
		address := cm.Data["ADDRESS"]
		return &address
	}
	return nil
}
