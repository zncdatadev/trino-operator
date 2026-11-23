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
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	opgoconfig "github.com/zncdatadev/operator-go/pkg/config"
	"github.com/zncdatadev/operator-go/pkg/constant"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	"github.com/zncdatadev/operator-go/pkg/security"
	"github.com/zncdatadev/operator-go/pkg/sidecar"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const (
	controllerTestNamespace      = "trino-test"
	controllerTestCluster        = "analytics"
	controllerTestGroup          = "default"
	controllerTestImage          = "example.invalid/trino:test"
	controllerTestQueryMaxMemory = "21GB"
	controllerTestLogLevel       = "debug"
	controllerTestLoggerLevel    = "WARN"
)

func TestTrinoRoleGroupHandler_DeclareRoles(t *testing.T) {
	tests := []struct {
		name            string
		tls             *trinov1alpha1.TlsSpec
		wantPortName    string
		wantPort        int32
		wantProbeScheme corev1.URIScheme
	}{
		{
			name:            "HTTP cluster",
			wantPortName:    trinov1alpha1.HttpPortName,
			wantPort:        trinov1alpha1.HttpPort,
			wantProbeScheme: corev1.URISchemeHTTP,
		},
		{
			name:            "TLS cluster",
			tls:             &trinov1alpha1.TlsSpec{},
			wantPortName:    trinov1alpha1.HttpsPortName,
			wantPort:        trinov1alpha1.HttpsPort,
			wantProbeScheme: corev1.URISchemeHTTPS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newControllerTestCluster()
			if tt.tls != nil {
				cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{Tls: tt.tls}
			}
			scheme := controllerTestScheme(t)
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			handler := NewTrinoRoleGroupHandler(scheme)

			catalog, err := handler.DeclareRoles(context.Background(), k8sClient, cr)
			if err != nil {
				t.Fatalf("DeclareRoles() error = %v", err)
			}
			if len(catalog) != 2 {
				t.Fatalf("DeclareRoles() returned %d roles, want 2", len(catalog))
			}

			for _, roleName := range []string{
				trinov1alpha1.TrinoCoordinatorRoleName,
				trinov1alpha1.TrinoWorkerRoleName,
			} {
				declaration, ok := catalog[roleName]
				if !ok {
					t.Fatalf("DeclareRoles() omitted role %q", roleName)
				}
				if declaration.MainContainerName != roleName {
					t.Errorf("%s main container = %q, want %q", roleName, declaration.MainContainerName, roleName)
				}
				if len(declaration.ContainerPorts) != 2 {
					t.Fatalf("%s container ports = %v, want application and metrics ports", roleName, declaration.ContainerPorts)
				}
				assertContainerPort(t, declaration.ContainerPorts, tt.wantPortName, tt.wantPort)
				assertContainerPort(t, declaration.ContainerPorts, trinov1alpha1.MetricsPortName, trinov1alpha1.MetricsPort)

				if len(declaration.ServicePorts) != 1 {
					t.Fatalf("%s service ports = %v, want only the application port", roleName, declaration.ServicePorts)
				}
				servicePort := declaration.ServicePorts[0]
				if servicePort.Name != tt.wantPortName || servicePort.Port != tt.wantPort ||
					servicePort.TargetPort.Type != intstr.String || servicePort.TargetPort.StrVal != tt.wantPortName {
					t.Errorf("%s application service port = %#v, want named %s:%d", roleName, servicePort, tt.wantPortName, tt.wantPort)
				}
				assertTrinoProbe(t, roleName+" readiness", declaration.ReadinessProbe, tt.wantPortName, tt.wantProbeScheme)
				assertTrinoProbe(t, roleName+" liveness", declaration.LivenessProbe, tt.wantPortName, tt.wantProbeScheme)

				if declaration.DataVolume == nil {
					t.Fatalf("%s data volume declaration is nil", roleName)
				}
				if declaration.DataVolume.Name != dataVolumeName || declaration.DataVolume.MountPath != constant.KubedoopDataDir {
					t.Errorf("%s data volume = %#v, want %q mounted at %q",
						roleName, declaration.DataVolume, dataVolumeName, constant.KubedoopDataDir)
				}
				if declaration.ConfigDefaults == nil || declaration.ConfigDefaults.Resources == nil ||
					declaration.ConfigDefaults.Resources.Storage == nil ||
					declaration.ConfigDefaults.Resources.Storage.Capacity == nil {
					t.Fatalf("%s does not declare the legacy data capacity default", roleName)
				}
				if got := declaration.ConfigDefaults.Resources.Storage.Capacity.String(); got != "1Gi" {
					t.Errorf("%s default data capacity = %q, want 1Gi", roleName, got)
				}

				if len(declaration.LogProducers) != 1 {
					t.Fatalf("%s log producers = %#v, want one Trino producer", roleName, declaration.LogProducers)
				}
				producer := declaration.LogProducers[0]
				if producer.Container != roleName || producer.Framework != "" ||
					producer.LogDirName != trinov1alpha1.DefaultProductName ||
					producer.LogFileName != roleName+".airlift.json" {
					t.Errorf("%s log producer = %#v, want product-rendered Airlift log in /kubedoop/log/trino", roleName, producer)
				}
				if declaration.LogVolumeSize != "1Gi" {
					t.Errorf("%s log volume size = %q, want 1Gi", roleName, declaration.LogVolumeSize)
				}
			}
		})
	}
}

func TestTrinoRoleGroupHandler_ResolveRoleGroupContributesBelowUserConfigOverrides(t *testing.T) {
	cr := newControllerTestCluster()
	cr.Spec.Coordinators.Config = &trinov1alpha1.ConfigSpec{
		QueryMaxMemory: controllerTestQueryMaxMemory,
	}
	cr.Spec.Coordinators.OverridesSpec = &commonsv1alpha1.OverridesSpec{
		ConfigOverrides: map[string]map[string]string{
			configFileName: {
				"query.max-memory":        "37GB",
				"http-server.log.enabled": stringTrue,
			},
			jvmFileName: {
				jvmMaxHeapOption: "3G",
			},
			logFileName: {
				"":                  controllerTestLogLevel,
				"io.trino.metadata": controllerTestLoggerLevel,
			},
		},
	}
	cr.Spec.Coordinators.RoleGroups[controllerTestGroup].OverridesSpec = &commonsv1alpha1.OverridesSpec{
		ConfigOverrides: map[string]map[string]string{
			configFileName: {
				"query.max-memory": "43GB",
			},
		},
	}

	_, buildCtx, _, contribution := prepareRoleGroupBuild(t, cr, trinov1alpha1.TrinoCoordinatorRoleName)
	contributed := contribution.ConfigOverrides[configFileName]
	if got := contributed["query.max-memory"]; got != controllerTestQueryMaxMemory {
		t.Fatalf("ResolveRoleGroup() contributed query.max-memory = %q, want typed role value 21GB", got)
	}
	if got := contributed["http-server.log.enabled"]; got != "false" {
		t.Fatalf("ResolveRoleGroup() contributed http-server.log.enabled = %q, want product value false", got)
	}

	merged := buildCtx.MergedConfig.ConfigFiles[configFileName]
	if got := merged["query.max-memory"]; got != "43GB" {
		t.Errorf("merged query.max-memory = %q, want role-group configOverride 43GB", got)
	}
	if got := merged["http-server.log.enabled"]; got != stringTrue {
		t.Errorf("merged http-server.log.enabled = %q, want role configOverride true", got)
	}
	if got := merged["discovery.uri"]; !strings.Contains(
		got,
		controllerTestCluster+"-coordinator."+controllerTestNamespace+".svc.cluster.local",
	) {
		t.Errorf("product-only discovery.uri = %q, want stable coordinator service", got)
	}
	if got := buildCtx.MergedConfig.ConfigFiles[jvmFileName][jvmMaxHeapOption]; got != "3G" {
		t.Errorf("merged jvm.config -Xmx = %q, want role configOverride 3G", got)
	}
	if got := buildCtx.MergedConfig.ConfigFiles[logFileName][""]; got != controllerTestLogLevel {
		t.Errorf("merged log.properties root level = %q, want role configOverride debug", got)
	}
	if got := buildCtx.MergedConfig.ConfigFiles[logFileName]["io.trino.metadata"]; got != controllerTestLoggerLevel {
		t.Errorf("merged log.properties io.trino.metadata = %q, want role configOverride WARN", got)
	}
}

func TestTrinoRoleGroupHandler_BuildResourcesRendersJVMAndLogOverrides(t *testing.T) {
	cr := newControllerTestCluster()
	cr.Spec.Workers.RoleGroups[controllerTestGroup].OverridesSpec = &commonsv1alpha1.OverridesSpec{
		ConfigOverrides: map[string]map[string]string{
			jvmFileName: {
				jvmMaxHeapOption: "3G",
				"-Dcustom.flag":  "enabled",
			},
			logFileName: {
				"":                  controllerTestLogLevel,
				"io.trino.metadata": controllerTestLoggerLevel,
			},
		},
	}
	handler, buildCtx, k8sClient, _ := prepareRoleGroupBuild(t, cr, trinov1alpha1.TrinoWorkerRoleName)

	resources, err := handler.BuildResources(context.Background(), k8sClient, cr, buildCtx)
	if err != nil {
		t.Fatalf("BuildResources() error = %v", err)
	}
	jvm := resources.ConfigMap.Data[jvmFileName]
	for _, line := range []string{jvmMaxHeapOption + "3G\n", "-Dcustom.flag=enabled\n"} {
		if !strings.Contains(jvm, line) {
			t.Errorf("generated jvm.config does not contain %q:\n%s", line, jvm)
		}
	}
	if strings.Count(jvm, jvmMaxHeapOption) != 1 {
		t.Errorf("generated jvm.config contains multiple max-heap options after override:\n%s", jvm)
	}

	properties, err := opgoconfig.NewPropertiesAdapter().Unmarshal(resources.ConfigMap.Data[logFileName])
	if err != nil {
		t.Fatalf("parse generated %s: %v", logFileName, err)
	}
	if properties[""] != controllerTestLogLevel || properties["io.trino.metadata"] != controllerTestLoggerLevel {
		t.Errorf("generated log.properties = %#v, want root debug and io.trino.metadata WARN", properties)
	}
}

func TestResolveAuthenticationRequiresClusterTLS(t *testing.T) {
	cr := newControllerTestCluster()
	cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
		Authentication: []trinov1alpha1.AuthenticationSpec{{
			AuthenticationClass: "oidc-auth",
			Oidc: &trinov1alpha1.OidcSpec{
				ClientCredentialsSecret: "oidc-client",
			},
		}},
	}

	_, err := resolveAuthentication(context.Background(), fake.NewClientBuilder().
		WithScheme(controllerTestScheme(t)).
		Build(), cr)
	if err == nil || !strings.Contains(err.Error(), "clusterConfig.tls") {
		t.Fatalf("resolveAuthentication() error = %v, want cluster TLS requirement", err)
	}
}

func TestTrinoRoleGroupHandler_BuildResourcesUsesCompatibleTLSClassesAndDistinctPaths(t *testing.T) {
	for _, roleName := range []string{
		trinov1alpha1.TrinoCoordinatorRoleName,
		trinov1alpha1.TrinoWorkerRoleName,
	} {
		t.Run(roleName, func(t *testing.T) {
			cr := newControllerTestCluster()
			cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{
				Tls: &trinov1alpha1.TlsSpec{
					ServerSecretClass:   "trino-server",
					InternalSecretClass: "trino-internal",
				},
			}
			handler, buildCtx, k8sClient, _ := prepareRoleGroupBuild(t, cr, roleName)

			resources, err := handler.BuildResources(context.Background(), k8sClient, cr, buildCtx)
			if err != nil {
				t.Fatalf("BuildResources() error = %v", err)
			}
			if resources.StatefulSet == nil || resources.ConfigMap == nil || resources.HeadlessService == nil {
				t.Fatal("BuildResources() omitted StatefulSet, ConfigMap or headless Service")
			}

			wantHeadlessService := controllerTestCluster + "-" + roleName + "-" + controllerTestGroup + "-headless"
			if resources.HeadlessService.Name != wantHeadlessService {
				t.Fatalf("headless Service name = %q, want %q", resources.HeadlessService.Name, wantHeadlessService)
			}
			serverScopeEntries := []string{
				string(security.PodScope),
				string(security.NodeScope),
				string(security.ServiceScope) + "=" + controllerTestCluster + "-" + roleName + "-" + controllerTestGroup,
				string(security.ServiceScope) + "=" + wantHeadlessService,
			}
			if roleName == trinov1alpha1.TrinoCoordinatorRoleName {
				serverScopeEntries = append(
					serverScopeEntries,
					string(security.ServiceScope)+"="+controllerTestCluster+"-"+trinov1alpha1.TrinoCoordinatorRoleName,
				)
			}
			wantServerScope := strings.Join(serverScopeEntries, security.CommonDelimiter)
			wantInternalScope := strings.Join(
				[]string{string(security.PodScope), string(security.NodeScope)},
				security.CommonDelimiter,
			)

			serverVolume := requireVolume(t, resources.StatefulSet.Spec.Template.Spec.Volumes, serverTLSVolumeName)
			internalVolume := requireVolume(t, resources.StatefulSet.Spec.Template.Spec.Volumes, internalTLSVolumeName)
			assertSecretClassVolume(t, serverVolume, "trino-server", wantServerScope)
			assertSecretClassVolume(t, internalVolume, "trino-server", wantInternalScope)

			container := requireContainer(t, resources.StatefulSet.Spec.Template.Spec.Containers, roleName)
			assertVolumeMount(t, container.VolumeMounts, serverTLSVolumeName, serverTLSMountPath)
			assertVolumeMount(t, container.VolumeMounts, internalTLSVolumeName, internalTLSMountPath)

			properties, err := opgoconfig.NewPropertiesAdapter().Unmarshal(resources.ConfigMap.Data[configFileName])
			if err != nil {
				t.Fatalf("parse generated %s: %v", configFileName, err)
			}
			wantPaths := map[string]string{
				"http-server.https.keystore.path":              serverTLSMountPath + "/keystore.p12",
				"http-server.https.truststore.path":            serverTLSMountPath + "/truststore.p12",
				"internal-communication.https.keystore.path":   internalTLSMountPath + "/keystore.p12",
				"internal-communication.https.truststore.path": internalTLSMountPath + "/truststore.p12",
			}
			for key, want := range wantPaths {
				if got := properties[key]; got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestTLSSecretClassesPreserveLegacyDefaulting(t *testing.T) {
	const customServerSecretClass = "company-ca"

	tests := []struct {
		name         string
		tls          *trinov1alpha1.TlsSpec
		wantServer   string
		wantInternal string
	}{
		{
			name:         "default classes",
			tls:          &trinov1alpha1.TlsSpec{},
			wantServer:   trinov1alpha1.DefaultTlsSecretClass,
			wantInternal: trinov1alpha1.DefaultTlsSecretClass,
		},
		{
			name: "legacy custom server with omitted internal class",
			tls: &trinov1alpha1.TlsSpec{
				ServerSecretClass: customServerSecretClass,
			},
			wantServer:   customServerSecretClass,
			wantInternal: customServerSecretClass,
		},
		{
			name: "legacy custom server with persisted internal default",
			tls: &trinov1alpha1.TlsSpec{
				ServerSecretClass:   customServerSecretClass,
				InternalSecretClass: trinov1alpha1.DefaultTlsSecretClass,
			},
			wantServer:   customServerSecretClass,
			wantInternal: customServerSecretClass,
		},
		{
			name: "legacy explicit distinct internal class remains dormant",
			tls: &trinov1alpha1.TlsSpec{
				ServerSecretClass:   customServerSecretClass,
				InternalSecretClass: "internal-ca",
			},
			wantServer:   customServerSecretClass,
			wantInternal: customServerSecretClass,
		},
		{
			name: "explicit internal class without server remains dormant",
			tls: &trinov1alpha1.TlsSpec{
				InternalSecretClass: "internal-ca",
			},
			wantServer:   trinov1alpha1.DefaultTlsSecretClass,
			wantInternal: trinov1alpha1.DefaultTlsSecretClass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newControllerTestCluster()
			cr.Spec.ClusterConfig = &trinov1alpha1.ClusterConfigSpec{Tls: tt.tls}

			serverClass, internalClass := tlsSecretClasses(cr)
			if serverClass != tt.wantServer || internalClass != tt.wantInternal {
				t.Fatalf(
					"tlsSecretClasses() = (%q, %q), want (%q, %q)",
					serverClass,
					internalClass,
					tt.wantServer,
					tt.wantInternal,
				)
			}
		})
	}
}

func TestTrinoRoleGroupHandler_BuildResourcesMetricsServiceTargetsNamedPort(t *testing.T) {
	cr := newControllerTestCluster()
	handler, buildCtx, k8sClient, _ := prepareRoleGroupBuild(t, cr, trinov1alpha1.TrinoWorkerRoleName)

	resources, err := handler.BuildResources(context.Background(), k8sClient, cr, buildCtx)
	if err != nil {
		t.Fatalf("BuildResources() error = %v", err)
	}
	if resources.MetricsService == nil {
		t.Fatal("BuildResources() omitted metrics Service")
	}
	if len(resources.MetricsService.Spec.Ports) != 1 {
		t.Fatalf("metrics Service ports = %#v, want one port", resources.MetricsService.Spec.Ports)
	}
	port := resources.MetricsService.Spec.Ports[0]
	if port.Port != trinov1alpha1.MetricsPort ||
		port.TargetPort.Type != intstr.String ||
		port.TargetPort.StrVal != trinov1alpha1.MetricsPortName {
		t.Errorf("metrics Service port = %#v, want %d -> named port %q",
			port, trinov1alpha1.MetricsPort, trinov1alpha1.MetricsPortName)
	}
	container := requireContainer(t, resources.StatefulSet.Spec.Template.Spec.Containers, trinov1alpha1.TrinoWorkerRoleName)
	assertContainerPort(t, container.Ports, trinov1alpha1.MetricsPortName, trinov1alpha1.MetricsPort)
}

func TestTrinoRoleGroupHandler_BuildResourcesPreservesEntrypointContracts(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(*testing.T, *trinov1alpha1.TrinoCluster)
		wantCommand []string
		wantArgs    []string
		wantScript  bool
	}{
		{
			name:        "default startup script",
			wantCommand: []string{"sh", "-c"},
			wantScript:  true,
		},
		{
			name: "cliOverrides replace the complete command",
			configure: func(_ *testing.T, cr *trinov1alpha1.TrinoCluster) {
				cr.Spec.Coordinators.RoleGroups[controllerTestGroup].OverridesSpec = &commonsv1alpha1.OverridesSpec{
					CliOverrides: []string{"/opt/trino/custom-entrypoint", "--safe-mode"},
				}
			},
			wantCommand: []string{"/opt/trino/custom-entrypoint", "--safe-mode"},
		},
		{
			name: "podOverrides entrypoint remains authoritative",
			configure: func(t *testing.T, cr *trinov1alpha1.TrinoCluster) {
				cr.Spec.Coordinators.RoleGroups[controllerTestGroup].OverridesSpec = &commonsv1alpha1.OverridesSpec{
					CliOverrides: []string{"/ignored-cli-entrypoint"},
					PodOverrides: mustRawPodOverride(t, &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:    trinov1alpha1.TrinoCoordinatorRoleName,
								Command: []string{"/pod-entrypoint"},
								Args:    []string{"--from-pod-override"},
							}},
						},
					}),
				}
			},
			wantCommand: []string{"/pod-entrypoint"},
			wantArgs:    []string{"--from-pod-override"},
		},
		{
			name: "unnamed podOverrides entrypoint remains authoritative",
			configure: func(t *testing.T, cr *trinov1alpha1.TrinoCluster) {
				cr.Spec.Coordinators.RoleGroups[controllerTestGroup].OverridesSpec = &commonsv1alpha1.OverridesSpec{
					PodOverrides: mustRawPodOverride(t, &corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Command: []string{"/unnamed-pod-entrypoint"},
								Args:    []string{"--from-unnamed-pod-override"},
							}},
						},
					}),
				}
			},
			wantCommand: []string{"/unnamed-pod-entrypoint"},
			wantArgs:    []string{"--from-unnamed-pod-override"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newControllerTestCluster()
			if tt.configure != nil {
				tt.configure(t, cr)
			}
			handler, buildCtx, k8sClient, _ := prepareRoleGroupBuild(t, cr, trinov1alpha1.TrinoCoordinatorRoleName)

			resources, err := handler.BuildResources(context.Background(), k8sClient, cr, buildCtx)
			if err != nil {
				t.Fatalf("BuildResources() error = %v", err)
			}
			container := requireContainer(t, resources.StatefulSet.Spec.Template.Spec.Containers, trinov1alpha1.TrinoCoordinatorRoleName)
			if !slices.Equal(container.Command, tt.wantCommand) {
				t.Errorf("container command = %q, want %q", container.Command, tt.wantCommand)
			}
			if tt.wantScript {
				if len(container.Args) != 1 {
					t.Fatalf("default container args = %q, want one startup script", container.Args)
				}
				if !strings.Contains(container.Args[0], "cp -RL "+constant.KubedoopConfigDirMount+"*") ||
					!strings.Contains(container.Args[0], "exec bin/launcher run") {
					t.Errorf("default startup script does not copy config and exec Trino:\n%s", container.Args[0])
				}
			} else if !slices.Equal(container.Args, tt.wantArgs) {
				t.Errorf("container args = %q, want %q", container.Args, tt.wantArgs)
			}
		})
	}
}

func newControllerTestCluster() *trinov1alpha1.TrinoCluster {
	replicas := int32(1)
	return &trinov1alpha1.TrinoCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: trinov1alpha1.GroupVersion.String(),
			Kind:       "TrinoCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerTestCluster,
			Namespace: controllerTestNamespace,
			UID:       types.UID("trino-test-uid"),
		},
		Spec: trinov1alpha1.TrinoClusterSpec{
			Image: &trinov1alpha1.ImageSpec{Custom: controllerTestImage},
			Coordinators: &trinov1alpha1.CoordinatorsSpec{
				RoleGroups: map[string]*trinov1alpha1.RoleGroupSpec{
					controllerTestGroup: {Replicas: ptr.To(replicas)},
				},
			},
			Workers: &trinov1alpha1.WorkersSpec{
				RoleGroups: map[string]*trinov1alpha1.RoleGroupSpec{
					controllerTestGroup: {Replicas: ptr.To(replicas)},
				},
			},
		},
	}
}

func controllerTestScheme(t *testing.T) *k8sruntime.Scheme {
	t.Helper()
	scheme := k8sruntime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core API to scheme: %v", err)
	}
	if err := authv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add authentication API to scheme: %v", err)
	}
	if err := trinov1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Trino API to scheme: %v", err)
	}
	return scheme
}

func prepareRoleGroupBuild(
	t *testing.T,
	cr *trinov1alpha1.TrinoCluster,
	roleName string,
) (*TrinoRoleGroupHandler, *reconciler.RoleGroupBuildContext, client.Client, *reconciler.Contribution) {
	t.Helper()
	scheme := controllerTestScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewTrinoRoleGroupHandler(scheme)

	catalog, err := handler.DeclareRoles(context.Background(), k8sClient, cr)
	if err != nil {
		t.Fatalf("DeclareRoles() error = %v", err)
	}
	declaration, ok := catalog[roleName]
	if !ok {
		t.Fatalf("DeclareRoles() omitted role %q", roleName)
	}

	clusterSpec := cr.GetSpec()
	roleSpec, ok := clusterSpec.Roles[roleName]
	if !ok {
		t.Fatalf("generic spec omitted role %q", roleName)
	}
	groupSpec, ok := roleSpec.RoleGroups[controllerTestGroup]
	if !ok {
		t.Fatalf("generic role %q omitted group %q", roleName, controllerTestGroup)
	}
	foldedConfig, _, err := reconciler.FoldCommonConfig(
		declaration.ConfigDefaults,
		roleSpec.GetConfig(),
		groupSpec.GetConfig(),
	)
	if err != nil {
		t.Fatalf("FoldCommonConfig() error = %v", err)
	}
	groupSpec.Config = foldedConfig

	buildCtx := &reconciler.RoleGroupBuildContext{
		ClusterName:      cr.Name,
		ClusterNamespace: cr.Namespace,
		ClusterLabels:    cr.Labels,
		ClusterSpec:      clusterSpec,
		RoleName:         roleName,
		RoleSpec:         &roleSpec,
		RoleGroupName:    controllerTestGroup,
		RoleGroupSpec:    groupSpec,
		ResourceName:     reconciler.RoleGroupResourceName(cr.Name, roleName, controllerTestGroup),
		SidecarManager:   sidecar.NewSidecarManager(),
		Declaration:      declaration,
		ResolvedImage: reconciler.ResolvedImage{
			Reference:      controllerTestImage,
			PullPolicy:     corev1.PullIfNotPresent,
			ProductVersion: trinov1alpha1.DefaultProductVersion,
		},
		ProductName: trinov1alpha1.DefaultProductName,
	}

	contribution, err := handler.ResolveRoleGroup(context.Background(), k8sClient, cr, buildCtx)
	if err != nil {
		t.Fatalf("ResolveRoleGroup() error = %v", err)
	}
	derived := contributionAsOverrides(t, contribution)
	buildCtx.MergedConfig = opgoconfig.NewConfigMerger().Merge(
		derived,
		roleSpec.GetOverrides(),
		groupSpec.GetOverrides(),
	)
	buildCtx.MergedConfig.Logging = foldedConfig.Logging
	return handler, buildCtx, k8sClient, contribution
}

func contributionAsOverrides(t *testing.T, contribution *reconciler.Contribution) *commonsv1alpha1.OverridesSpec {
	t.Helper()
	if contribution == nil {
		return nil
	}
	result := &commonsv1alpha1.OverridesSpec{
		ConfigOverrides: contribution.ConfigOverrides,
		EnvOverrides:    contribution.EnvVars,
	}
	if contribution.PodOverrides != nil {
		raw, err := json.Marshal(contribution.PodOverrides)
		if err != nil {
			t.Fatalf("marshal contributed pod override: %v", err)
		}
		result.PodOverrides = &k8sruntime.RawExtension{Raw: raw}
	}
	return result
}

func mustRawPodOverride(t *testing.T, pod *corev1.PodTemplateSpec) *k8sruntime.RawExtension {
	t.Helper()
	raw, err := json.Marshal(pod)
	if err != nil {
		t.Fatalf("marshal pod override: %v", err)
	}
	return &k8sruntime.RawExtension{Raw: raw}
}

func assertContainerPort(t *testing.T, ports []corev1.ContainerPort, name string, number int32) {
	t.Helper()
	for _, port := range ports {
		if port.Name == name {
			if port.ContainerPort != number || port.Protocol != corev1.ProtocolTCP {
				t.Errorf("container port %q = %#v, want TCP/%d", name, port, number)
			}
			return
		}
	}
	t.Errorf("container ports %#v do not include %q", ports, name)
}

func assertTrinoProbe(t *testing.T, label string, probe *corev1.Probe, portName string, scheme corev1.URIScheme) {
	t.Helper()
	if probe == nil || probe.HTTPGet == nil {
		t.Fatalf("%s probe is not an HTTP probe: %#v", label, probe)
	}
	if probe.HTTPGet.Path != "/v1/info" ||
		probe.HTTPGet.Port.Type != intstr.String ||
		probe.HTTPGet.Port.StrVal != portName ||
		probe.HTTPGet.Scheme != scheme {
		t.Errorf("%s probe = %#v, want %s /v1/info on named port %q", label, probe.HTTPGet, scheme, portName)
	}
}

func requireVolume(t *testing.T, volumes []corev1.Volume, name string) corev1.Volume {
	t.Helper()
	for _, volume := range volumes {
		if volume.Name == name {
			return volume
		}
	}
	t.Fatalf("volumes %#v do not include %q", volumes, name)
	return corev1.Volume{}
}

func assertSecretClassVolume(t *testing.T, volume corev1.Volume, wantClass, wantScope string) {
	t.Helper()
	if volume.Ephemeral == nil || volume.Ephemeral.VolumeClaimTemplate == nil {
		t.Fatalf("volume %q is not an ephemeral SecretClass volume: %#v", volume.Name, volume)
	}
	template := volume.Ephemeral.VolumeClaimTemplate
	if got := template.Annotations[security.SecretClassAnnotation]; got != wantClass {
		t.Errorf("volume %q SecretClass = %q, want %q", volume.Name, got, wantClass)
	}
	if got := template.Annotations[security.SecretClassScopeAnnotation]; got != wantScope {
		t.Errorf("volume %q scope = %q, want %q", volume.Name, got, wantScope)
	}
	if got := template.Annotations[security.AnnotationSecretsFormat]; got != string(security.TLSP12) {
		t.Errorf("volume %q format = %q, want %q", volume.Name, got, security.TLSP12)
	}
	if got := template.Spec.Resources.Requests[corev1.ResourceStorage]; got.String() != "1Mi" {
		t.Errorf("volume %q storage = %q, want 1Mi", volume.Name, got.String())
	}
}

func assertVolumeMount(t *testing.T, mounts []corev1.VolumeMount, name, path string) {
	t.Helper()
	for _, mount := range mounts {
		if mount.Name == name {
			if mount.MountPath != path {
				t.Errorf("volume mount %q path = %q, want %q", name, mount.MountPath, path)
			}
			return
		}
	}
	t.Errorf("volume mounts %#v do not include %q", mounts, name)
}

func requireContainer(t *testing.T, containers []corev1.Container, name string) *corev1.Container {
	t.Helper()
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	t.Fatalf("containers %#v do not include %q", containers, name)
	return nil
}
