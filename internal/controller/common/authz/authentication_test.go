package authz

import (
	"context"
	"reflect"
	"strings"
	"testing"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/constant"
	"github.com/zncdatadev/operator-go/pkg/security"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const (
	testOIDCHostname       = "identity.example.com"
	testOIDCPrincipalClaim = "sub"
	testLDAPSecretClass    = "ldap-bind"
	testOIDCClientSecret   = "oidc-client"
	testOIDCGroupsScope    = "groups"
	testOIDCWebPKIPath     = "server.caCert.webPki"
	testOIDCRootPath       = "/realms/trino"
	testOIDCIPv6Hostname   = "2001:db8::1"
	testExactlyOneProvider = "exactly one provider"
)

func TestNewAuthenticationCreatesOIDCAuthenticator(t *testing.T) {
	oidcClass := testAuthenticationClass("oidc-auth", &authv1alpha1.AuthenticationProvider{
		OIDC: &authv1alpha1.OIDCProvider{
			Hostname:       testOIDCHostname,
			Port:           8080,
			PrincipalClaim: testOIDCPrincipalClaim,
			RootPath:       "/realms/trino",
		},
	})
	client := testClient(t, oidcClass)
	specs := []trinov1alpha1.AuthenticationSpec{
		{
			AuthenticationClass: oidcClass.Name,
			Oidc: &trinov1alpha1.OidcSpec{
				ClientCredentialsSecret: "trino-oidc-client",
				ExtraScopes:             []string{testOIDCGroupsScope},
			},
		},
	}

	authentication, err := NewAuthentication(context.Background(), client, "trino", specs)
	if err != nil {
		t.Fatalf("NewAuthentication() error = %v", err)
	}
	if len(authentication.Authenticators) != 1 {
		t.Fatalf("got %d authenticators, want 1", len(authentication.Authenticators))
	}

	oidc, ok := authentication.Authenticators[0].(*Oidc)
	if !ok || oidc.AuthenticationClassName != oidcClass.Name {
		t.Fatalf("authenticator = %#v, want OIDC class %q", authentication.Authenticators[0], oidcClass.Name)
	}
	if got := authentication.GetConfigProperties()[authenticationTypeProperty]; got != oidcAuthenticationType {
		t.Fatalf("authentication type = %q, want OAUTH2", got)
	}
}

func TestAuthenticatorFactoryRequiresExactlyOneProvider(t *testing.T) {
	config := &trinov1alpha1.OidcSpec{ClientCredentialsSecret: testOIDCClientSecret}
	tests := []struct {
		name          string
		provider      *authv1alpha1.AuthenticationProvider
		wantFragments []string
	}{
		{
			name:          "nil provider",
			wantFragments: []string{"no provider", testExactlyOneProvider},
		},
		{
			name:          "empty provider",
			provider:      &authv1alpha1.AuthenticationProvider{},
			wantFragments: []string{"no provider", testExactlyOneProvider},
		},
		{
			name: "OIDC and LDAP",
			provider: &authv1alpha1.AuthenticationProvider{
				OIDC: &authv1alpha1.OIDCProvider{
					Hostname:       testOIDCHostname,
					PrincipalClaim: testOIDCPrincipalClaim,
				},
				LDAP: testLDAPProvider(),
			},
			wantFragments: []string{"multiple providers", string(AuthenticationTypeOIDC), string(AuthenticationTypeLDAP), testExactlyOneProvider},
		},
		{
			name: "TLS, static, and Kerberos",
			provider: &authv1alpha1.AuthenticationProvider{
				TLS:      &authv1alpha1.TLSProvider{},
				Static:   &authv1alpha1.StaticProvider{},
				Kerberos: &authv1alpha1.KerberosProvider{},
			},
			wantFragments: []string{"multiple providers", string(AuthenticationTypeTls), string(AuthenticationTypeStatic), "kerberos", testExactlyOneProvider},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := authenticatorFactory("ambiguous-auth", config, test.provider)
			if err == nil {
				t.Fatal("authenticatorFactory() error = nil, want provider cardinality error")
			}
			for _, fragment := range test.wantFragments {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("authenticatorFactory() error = %q, want %q", err, fragment)
				}
			}
		})
	}
}

func TestNewAuthenticationRejectsUnsupportedProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider *authv1alpha1.AuthenticationProvider
		want     string
	}{
		{
			name: "static",
			provider: &authv1alpha1.AuthenticationProvider{
				Static: &authv1alpha1.StaticProvider{
					UserCredentialsSecret: &authv1alpha1.StaticCredentialsSecret{Name: "users"},
				},
			},
			want: "static authentication",
		},
		{
			name: "LDAP",
			provider: &authv1alpha1.AuthenticationProvider{
				LDAP: testLDAPProvider(),
			},
			want: "LDAP authentication",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authClass := testAuthenticationClass(test.name+"-auth", test.provider)
			_, err := NewAuthentication(context.Background(), testClient(t, authClass), "trino", []trinov1alpha1.AuthenticationSpec{{
				AuthenticationClass: authClass.Name,
			}})
			if err == nil {
				t.Fatal("NewAuthentication() error = nil, want unsupported provider error")
			}
			if !strings.Contains(err.Error(), authClass.Name) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("NewAuthentication() error = %q, want class and %q context", err, test.want)
			}
		})
	}
}

func TestNewAuthenticationReturnsMissingClassError(t *testing.T) {
	client := testClient(t)

	_, err := NewAuthentication(context.Background(), client, "trino", []trinov1alpha1.AuthenticationSpec{{
		AuthenticationClass: "missing-auth",
	}})
	if err == nil {
		t.Fatal("NewAuthentication() error = nil, want missing AuthenticationClass error")
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("NewAuthentication() error = %v, want Kubernetes NotFound error", err)
	}
	if !strings.Contains(err.Error(), "missing-auth") || !strings.Contains(err.Error(), "trino") {
		t.Fatalf("NewAuthentication() error = %q, want class and namespace context", err)
	}
}

func TestNewAuthenticationRejectsDuplicateProviderType(t *testing.T) {
	first := testAuthenticationClass("first-oidc", &authv1alpha1.AuthenticationProvider{
		OIDC: &authv1alpha1.OIDCProvider{Hostname: "first.example.com", PrincipalClaim: testOIDCPrincipalClaim},
	})
	second := testAuthenticationClass("second-oidc", &authv1alpha1.AuthenticationProvider{
		OIDC: &authv1alpha1.OIDCProvider{Hostname: "second.example.com", PrincipalClaim: testOIDCPrincipalClaim},
	})

	_, err := NewAuthentication(context.Background(), testClient(t, first, second), "trino", []trinov1alpha1.AuthenticationSpec{
		{AuthenticationClass: first.Name, Oidc: &trinov1alpha1.OidcSpec{ClientCredentialsSecret: "first-client"}},
		{AuthenticationClass: second.Name, Oidc: &trinov1alpha1.OidcSpec{ClientCredentialsSecret: "second-client"}},
	})
	if err == nil {
		t.Fatal("NewAuthentication() error = nil, want duplicate OIDC provider error")
	}
	for _, fragment := range []string{"oidc", first.Name, second.Name} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("NewAuthentication() error = %q, want %q", err, fragment)
		}
	}
}

func TestNewAuthenticationRejectsNilOIDCConfig(t *testing.T) {
	authClass := testAuthenticationClass("oidc-auth", &authv1alpha1.AuthenticationProvider{
		OIDC: &authv1alpha1.OIDCProvider{Hostname: testOIDCHostname, PrincipalClaim: testOIDCPrincipalClaim},
	})

	_, err := NewAuthentication(context.Background(), testClient(t, authClass), "trino", []trinov1alpha1.AuthenticationSpec{{
		AuthenticationClass: authClass.Name,
	}})
	if err == nil {
		t.Fatal("NewAuthentication() error = nil, want missing OIDC configuration error")
	}
	if !strings.Contains(err.Error(), authClass.Name) || !strings.Contains(err.Error(), "OIDC configuration") {
		t.Fatalf("NewAuthentication() error = %q, want class and OIDC configuration context", err)
	}
}

func TestStaticZeroValueIsSafeAndReturnsProperties(t *testing.T) {
	static := &Static{}

	properties := static.GetConfigProperties()
	if properties == nil {
		t.Fatal("Static.GetConfigProperties() = nil, want empty map")
	}
	if len(properties) != 0 {
		t.Fatalf("Static.GetConfigProperties() = %#v, want an empty map until password files are supported", properties)
	}

	// A malformed legacy AuthenticationClass must not turn property or volume aggregation into a panic.
	_ = static.GetCommands()
	_ = static.GetEnvVars()
	_ = static.GetVolumeMounts()
	_ = static.GetVolumes()

	var nilStatic *Static
	_ = nilStatic.GetCommands()
	_ = nilStatic.GetConfigProperties()
	_ = nilStatic.GetEnvVars()
	_ = nilStatic.GetVolumeMounts()
	_ = nilStatic.GetVolumes()
}

func TestLDAPVolumeUsesV013SecurityContract(t *testing.T) {
	ldap := &Ldap{
		AuthenticationClassName: "ldap-auth",
		Provider: &authv1alpha1.LDAPProvider{
			BindCredentials: &commonsv1alpha1.Credentials{
				SecretClass: testLDAPSecretClass,
				Scope: &commonsv1alpha1.CredentialsScope{
					Node:            true,
					Pod:             true,
					Services:        []string{"trino"},
					ListenerVolumes: []string{"client"},
				},
			},
		},
	}

	volumes := ldap.GetVolumes()
	if len(volumes) != 1 || volumes[0].Ephemeral == nil || volumes[0].Ephemeral.VolumeClaimTemplate == nil {
		t.Fatalf("LDAP volumes = %#v, want one ephemeral PVC volume", volumes)
	}
	pvc := volumes[0].Ephemeral.VolumeClaimTemplate
	wantAnnotations := map[string]string{
		security.SecretClassAnnotation:      testLDAPSecretClass,
		security.SecretClassScopeAnnotation: "node,pod,service=trino,listener-volume=client",
	}
	if !reflect.DeepEqual(pvc.Annotations, wantAnnotations) {
		t.Fatalf("LDAP PVC annotations = %#v, want %#v", pvc.Annotations, wantAnnotations)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != string(security.SecretStorageClass) {
		t.Fatalf("LDAP PVC storage class = %v, want %q", pvc.Spec.StorageClassName, security.SecretStorageClass)
	}

	mounts := ldap.GetVolumeMounts()
	wantMountPath := constant.KubedoopTlsDir + testLDAPSecretClass
	if len(mounts) != 1 || mounts[0].MountPath != wantMountPath {
		t.Fatalf("LDAP mounts = %#v, want path %q", mounts, wantMountPath)
	}
}

func testAuthenticationClass(name string, provider *authv1alpha1.AuthenticationProvider) *authv1alpha1.AuthenticationClass {
	return &authv1alpha1.AuthenticationClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: authv1alpha1.AuthenticationClassSpec{
			AuthenticationProvider: provider,
		},
	}
}

func testLDAPProvider() *authv1alpha1.LDAPProvider {
	return &authv1alpha1.LDAPProvider{
		BindCredentials: &commonsv1alpha1.Credentials{SecretClass: testLDAPSecretClass},
		Hostname:        "ldap.example.com",
		LDAPFieldNames:  &authv1alpha1.LDAPFieldNames{Uid: "uid"},
		SearchBase:      "ou=users,dc=example,dc=com",
	}
}

func testClient(t *testing.T, objects ...ctrlclient.Object) ctrlclient.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := authv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register authentication API: %v", err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}
