/*
Copyright 2023 zncdatadev.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1_test

import (
	"encoding/json"
	"reflect"
	"testing"

	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/listener"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const testRoleGroupName = "default"

func TestGetSpecProjectsTypedRoles(t *testing.T) {
	t.Parallel()

	zero := int32(0)
	two := int32(2)
	maxUnavailable := int32(1)
	enabled := true
	roleTimeout := "45s"
	groupTimeout := "90s"
	operation := &commonsv1alpha1.ClusterOperationSpec{
		ReconciliationPaused: true,
		Stopped:              true,
	}
	coordinatorRoleConfig := &commonsv1alpha1.RoleConfigSpec{
		PodDisruptionBudget: &commonsv1alpha1.PodDisruptionBudgetSpec{
			Enabled:        &enabled,
			MaxUnavailable: &maxUnavailable,
		},
	}
	coordinatorConfig := &commonsv1alpha1.RoleGroupConfigSpec{GracefulShutdownTimeout: &roleTimeout}
	coordinatorGroupConfig := &commonsv1alpha1.RoleGroupConfigSpec{GracefulShutdownTimeout: &groupTimeout}
	workerConfig := &commonsv1alpha1.RoleGroupConfigSpec{}

	coordinatorRoleOverrides := overrides("coordinator-role")
	coordinatorGroupOverrides := overrides("coordinator-group")
	workerRoleOverrides := overrides("worker-role")
	workerGroupOverrides := overrides("worker-group")

	cluster := &trinov1alpha1.TrinoCluster{
		Spec: trinov1alpha1.TrinoClusterSpec{
			ClusterOperation: operation,
			Image: &trinov1alpha1.ImageSpec{
				Custom:          "registry.example/trino:custom",
				Repo:            "registry.example/products",
				ProductVersion:  "451",
				KubedoopVersion: "0.13.0",
				PullPolicy:      corev1.PullAlways,
				PullSecretName:  "registry-credentials",
			},
			Coordinators: &trinov1alpha1.CoordinatorsSpec{
				BaseRoleSpec: trinov1alpha1.BaseRoleSpec{
					OverridesSpec: coordinatorRoleOverrides,
					RoleConfig:    coordinatorRoleConfig,
					Config: &trinov1alpha1.ConfigSpec{
						RoleGroupConfigSpec: coordinatorConfig,
						QueryMaxMemory:      "20GB",
					},
				},
				RoleGroups: map[string]*trinov1alpha1.RoleGroupSpec{
					testRoleGroupName: {
						Replicas:      &zero,
						OverridesSpec: coordinatorGroupOverrides,
						Config: &trinov1alpha1.ConfigSpec{
							RoleGroupConfigSpec:   coordinatorGroupConfig,
							QueryMaxMemoryPerNode: "2GB",
						},
					},
					"ignored-nil": nil,
				},
			},
			Workers: &trinov1alpha1.WorkersSpec{
				BaseRoleSpec: trinov1alpha1.BaseRoleSpec{
					OverridesSpec: workerRoleOverrides,
					Config:        &trinov1alpha1.ConfigSpec{RoleGroupConfigSpec: workerConfig},
				},
				RoleGroups: map[string]*trinov1alpha1.RoleGroupSpec{
					testRoleGroupName: {
						Replicas:      &two,
						OverridesSpec: workerGroupOverrides,
					},
				},
			},
		},
	}

	got := cluster.GetSpec()
	if got == nil {
		t.Fatal("GetSpec returned nil")
	}
	if got.ClusterOperation != operation {
		t.Fatal("cluster operation pointer was not preserved")
	}
	wantImage := &commonsv1alpha1.ImageSpec{
		Custom:          "registry.example/trino:custom",
		Repo:            "registry.example/products",
		ProductVersion:  "451",
		KubedoopVersion: "0.13.0",
		PullPolicy:      corev1.PullAlways,
		PullSecretName:  "registry-credentials",
	}
	if !reflect.DeepEqual(got.Image, wantImage) {
		t.Fatalf("image projection mismatch:\n got: %#v\nwant: %#v", got.Image, wantImage)
	}

	if len(got.Roles) != 2 {
		t.Fatalf("got %d roles, want 2: %#v", len(got.Roles), got.Roles)
	}
	if _, exists := got.Roles["coordinators"]; exists {
		t.Fatal("plural coordinators role key would change existing resource names")
	}
	if _, exists := got.Roles["workers"]; exists {
		t.Fatal("plural workers role key would change existing resource names")
	}

	coordinator := got.Roles[trinov1alpha1.TrinoCoordinatorRoleName]
	if coordinator.RoleConfig != coordinatorRoleConfig {
		t.Fatal("coordinator roleConfig pointer was not preserved")
	}
	if coordinator.Config != coordinatorConfig {
		t.Fatal("coordinator common config pointer was not preserved")
	}
	assertOverrides(t, "coordinator role", coordinatorRoleOverrides, coordinator.GetOverrides())
	if len(coordinator.RoleGroups) != 1 {
		t.Fatalf("nil coordinator role group should be ignored, got %#v", coordinator.RoleGroups)
	}
	coordinatorGroup := coordinator.RoleGroups[testRoleGroupName]
	if coordinatorGroup.Replicas != &zero || coordinatorGroup.GetReplicas() != 0 {
		t.Fatalf("explicit zero replicas was defaulted or copied incorrectly: %#v", coordinatorGroup.Replicas)
	}
	if coordinatorGroup.Config != coordinatorGroupConfig {
		t.Fatal("coordinator role group common config pointer was not preserved")
	}
	assertOverrides(t, "coordinator role group", coordinatorGroupOverrides, coordinatorGroup.GetOverrides())

	worker := got.Roles[trinov1alpha1.TrinoWorkerRoleName]
	if worker.Config != workerConfig {
		t.Fatal("worker common config pointer was not preserved")
	}
	assertOverrides(t, "worker role", workerRoleOverrides, worker.GetOverrides())
	workerGroup := worker.RoleGroups[testRoleGroupName]
	if workerGroup.Replicas != &two || workerGroup.GetReplicas() != 2 {
		t.Fatalf("worker replicas mismatch: %#v", workerGroup.Replicas)
	}
	assertOverrides(t, "worker role group", workerGroupOverrides, workerGroup.GetOverrides())
}

func TestGetSpecHandlesNilOptionalFields(t *testing.T) {
	t.Parallel()

	cluster := &trinov1alpha1.TrinoCluster{}
	got := cluster.GetSpec()
	if got == nil {
		t.Fatal("GetSpec returned nil for a zero-value cluster")
	}
	if got.Image != nil || got.ClusterOperation != nil {
		t.Fatalf("nil optional fields were materialized: %#v", got)
	}
	if got.Roles == nil || len(got.Roles) != 0 {
		t.Fatalf("zero-value cluster should project an empty roles map, got %#v", got.Roles)
	}

	cluster.Spec.Coordinators = &trinov1alpha1.CoordinatorsSpec{
		BaseRoleSpec: trinov1alpha1.BaseRoleSpec{
			// A product-only leaf does not manufacture an empty generic common config.
			Config: &trinov1alpha1.ConfigSpec{QueryMaxMemory: "12GB"},
		},
	}
	coordinator := cluster.GetSpec().Roles[trinov1alpha1.TrinoCoordinatorRoleName]
	if coordinator.Config != nil {
		t.Fatalf("nil embedded common config was materialized: %#v", coordinator.Config)
	}
	if coordinator.RoleGroups != nil {
		t.Fatalf("nil roleGroups was materialized: %#v", coordinator.RoleGroups)
	}
}

func TestGetStatusReturnsEmbeddedStatus(t *testing.T) {
	t.Parallel()

	cluster := &trinov1alpha1.TrinoCluster{
		Status: trinov1alpha1.TrinoClusterStatus{
			URLs:       []trinov1alpha1.StatusURL{{Name: "ui", URL: "https://trino.example"}},
			Generation: 3,
			Name:       "example",
			Type:       "TrinoCluster",
		},
	}

	status := cluster.GetStatus()
	if status != &cluster.Status.GenericClusterStatus {
		t.Fatal("GetStatus did not return a pointer into the cluster object")
	}
	status.SetObservedGeneration(7)
	status.SetAvailable("Ready", "all role groups are ready")
	status.RoleGroups = map[string][]string{
		trinov1alpha1.TrinoCoordinatorRoleName: {testRoleGroupName},
	}
	if cluster.Status.ObservedGeneration != 7 || len(cluster.Status.Conditions) != 1 {
		t.Fatalf("writes through GetStatus were not retained: %#v", cluster.Status)
	}

	encoded, err := json.Marshal(cluster.Status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal status fields: %v", err)
	}
	for _, key := range []string{"conditions", "roleGroups", "observedGeneration", "urls", "generation", "name", "type"} {
		if _, exists := fields[key]; !exists {
			t.Errorf("status JSON lost field %q: %s", key, encoded)
		}
	}

	clone := cluster.DeepCopy()
	clone.Status.URLs[0].URL = "https://changed.example"
	clone.Status.Conditions[0].Message = "changed"
	clone.Status.RoleGroups[trinov1alpha1.TrinoCoordinatorRoleName][0] = "changed"
	if cluster.Status.URLs[0].URL != "https://trino.example" ||
		cluster.Status.Conditions[0].Message != "all role groups are ready" ||
		cluster.Status.RoleGroups[trinov1alpha1.TrinoCoordinatorRoleName][0] != testRoleGroupName {
		t.Fatal("TrinoCluster deepcopy aliases status storage")
	}
}

func TestVectorAggregatorConfigMapName(t *testing.T) {
	t.Parallel()

	cluster := &trinov1alpha1.TrinoCluster{}
	if got := cluster.VectorAggregatorConfigMapName(); got != "" {
		t.Fatalf("nil clusterConfig returned %q, want empty", got)
	}
	cluster.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
		VectorAggregatorConfigMapName: "vector-discovery",
	}
	if got := cluster.VectorAggregatorConfigMapName(); got != "vector-discovery" {
		t.Fatalf("VectorAggregatorConfigMapName() = %q, want vector-discovery", got)
	}
}

func TestListenerClassCompatibility(t *testing.T) {
	t.Parallel()

	if trinov1alpha1.DefaultListenerClass != listener.ListenerClassClusterInternal {
		t.Fatalf("default listener class = %q, want cluster-internal", trinov1alpha1.DefaultListenerClass)
	}
	config := trinov1alpha1.ClusterConfigSpec{ListenerClass: listener.ListenerClassExternalStable}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal listener class: %v", err)
	}
	if string(encoded) != `{"listenerClass":"external-stable"}` {
		t.Fatalf("listener class YAML/JSON shape changed: %s", encoded)
	}
}

func overrides(prefix string) *commonsv1alpha1.OverridesSpec {
	return &commonsv1alpha1.OverridesSpec{
		ConfigOverrides: map[string]map[string]string{
			"config.properties": {"source": prefix},
		},
		EnvOverrides: map[string]string{"SOURCE": prefix},
		CliOverrides: []string{"--source=" + prefix},
		PodOverrides: &runtime.RawExtension{Raw: []byte(`{"metadata":{"labels":{"source":"` + prefix + `"}}}`)},
	}
}

func assertOverrides(t *testing.T, scope string, want, got *commonsv1alpha1.OverridesSpec) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s overrides mismatch:\n got: %#v\nwant: %#v", scope, got, want)
	}
}
