/*
Copyright 2026 zncdatadev.

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

package controller

import (
	"fmt"
	"strings"
	"testing"

	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

func TestDiscoveryURIUsesStableCoordinatorService(t *testing.T) {
	tests := []struct {
		name       string
		tls        bool
		wantScheme string
		wantPort   int32
	}{
		{
			name:       "HTTP",
			wantScheme: httpScheme,
			wantPort:   trinov1alpha1.HttpPort,
		},
		{
			name:       "TLS",
			tls:        true,
			wantScheme: httpsScheme,
			wantPort:   trinov1alpha1.HttpsPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disabledReplicas := int32(0)
			activeReplicas := int32(2)
			cr := newControllerTestCluster()
			cr.Spec.Coordinators.RoleGroups = map[string]*trinov1alpha1.RoleGroupSpec{
				"a-disabled": {Replicas: &disabledReplicas},
				"z-active":   {Replicas: &activeReplicas},
			}
			if tt.tls {
				cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
					Tls: &trinov1alpha1.TlsSpec{},
				}
			}

			want := fmt.Sprintf(
				"%s://%s-%s.%s.svc.cluster.local:%d",
				tt.wantScheme,
				controllerTestCluster,
				trinov1alpha1.TrinoCoordinatorRoleName,
				controllerTestNamespace,
				tt.wantPort,
			)
			if got := discoveryURI(cr); got != want {
				t.Errorf("discoveryURI() = %q, want %q", got, want)
			}
		})
	}
}

func TestJVMConfigAdapterRejectsMultilineOptions(t *testing.T) {
	_, err := (jvmConfigAdapter{}).Marshal(map[string]string{
		"-Dsafe": "true\n-Dinjected=true",
	})
	if err == nil || !strings.Contains(err.Error(), "newline") {
		t.Fatalf("Marshal() error = %v, want newline validation error", err)
	}
}

func TestResolveQueryMemory(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		configure   func(*trinov1alpha1.TrinoCluster)
		wantMax     string
		wantPerNode string
		wantInclude bool
	}{
		{
			name:        "legacy defaults remain when no config exists",
			roleName:    trinov1alpha1.TrinoCoordinatorRoleName,
			wantMax:     trinov1alpha1.DefaultQueryMaxMemory,
			wantPerNode: defaultQueryMaxPerNode,
			wantInclude: false,
		},
		{
			name:     "role leaves survive an unrelated role-group config block",
			roleName: trinov1alpha1.TrinoCoordinatorRoleName,
			configure: func(cr *trinov1alpha1.TrinoCluster) {
				cr.Spec.Coordinators.Config = &trinov1alpha1.ConfigSpec{
					QueryMaxMemory:        controllerTestQueryMaxMemory,
					QueryMaxMemoryPerNode: "2GB",
				}
				cr.Spec.Coordinators.RoleGroups[controllerTestGroup].Config = &trinov1alpha1.ConfigSpec{
					RoleGroupConfigSpec: &commonsv1alpha1.RoleGroupConfigSpec{
						Logging: &commonsv1alpha1.LoggingSpec{},
					},
				}
			},
			wantMax:     controllerTestQueryMaxMemory,
			wantPerNode: "2GB",
			wantInclude: true,
		},
		{
			name:     "role group overrides one leaf and inherits the other",
			roleName: trinov1alpha1.TrinoWorkerRoleName,
			configure: func(cr *trinov1alpha1.TrinoCluster) {
				cr.Spec.Workers.Config = &trinov1alpha1.ConfigSpec{
					QueryMaxMemory:        "19GB",
					QueryMaxMemoryPerNode: "1900MB",
				}
				cr.Spec.Workers.RoleGroups[controllerTestGroup].Config = &trinov1alpha1.ConfigSpec{
					QueryMaxMemory: "31GB",
				}
			},
			wantMax:     "31GB",
			wantPerNode: "1900MB",
			wantInclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newControllerTestCluster()
			if tt.configure != nil {
				tt.configure(cr)
			}

			gotMax, gotPerNode, gotInclude := resolveQueryMemory(cr, tt.roleName, controllerTestGroup)
			if gotMax != tt.wantMax {
				t.Errorf("query.max-memory = %q, want %q", gotMax, tt.wantMax)
			}
			if gotPerNode != tt.wantPerNode {
				t.Errorf("query.max-memory-per-node = %q, want %q", gotPerNode, tt.wantPerNode)
			}
			if gotInclude != tt.wantInclude {
				t.Errorf("include query.max-memory-per-node = %t, want %t", gotInclude, tt.wantInclude)
			}
		})
	}
}
