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

package authz

import (
	"strings"
	"testing"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	commonsv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/commons/v1alpha1"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

func TestOidcIssuerUsesTLSAndOmitsOnlyTheSchemeDefaultPort(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		tls      *authv1alpha1.OIDCTls
		port     int
		wantURL  string
	}{
		{
			name:     "HTTP omitted port",
			hostname: testOIDCHostname,
			wantURL:  "http://identity.example.com/realms/trino",
		},
		{
			name:     "HTTP default port",
			hostname: testOIDCHostname,
			port:     oidcDefaultHTTPPort,
			wantURL:  "http://identity.example.com/realms/trino",
		},
		{
			name:     "HTTP non-default port",
			hostname: testOIDCHostname,
			port:     8080,
			wantURL:  "http://identity.example.com:8080/realms/trino",
		},
		{
			name:     "HTTPS omitted port",
			hostname: testOIDCHostname,
			tls:      testWebPKIOIDCTLS(),
			wantURL:  "https://identity.example.com/realms/trino",
		},
		{
			name:     "HTTPS default port",
			hostname: testOIDCHostname,
			tls:      testWebPKIOIDCTLS(),
			port:     oidcDefaultHTTPSPort,
			wantURL:  "https://identity.example.com/realms/trino",
		},
		{
			name:     "HTTPS non-default port",
			hostname: testOIDCHostname,
			tls:      testWebPKIOIDCTLS(),
			port:     8443,
			wantURL:  "https://identity.example.com:8443/realms/trino",
		},
		{
			name:     "bare IPv6 omitted port",
			hostname: testOIDCIPv6Hostname,
			wantURL:  "http://[2001:db8::1]/realms/trino",
		},
		{
			name:     "bare IPv6 non-default port",
			hostname: testOIDCIPv6Hostname,
			port:     8080,
			wantURL:  "http://[2001:db8::1]:8080/realms/trino",
		},
		{
			name:     "bracketed IPv6 default port",
			hostname: "[" + testOIDCIPv6Hostname + "]",
			tls:      testWebPKIOIDCTLS(),
			port:     oidcDefaultHTTPSPort,
			wantURL:  "https://[2001:db8::1]/realms/trino",
		},
		{
			name:     "bracketed IPv6 non-default port",
			hostname: "[" + testOIDCIPv6Hostname + "]",
			tls:      testWebPKIOIDCTLS(),
			port:     8443,
			wantURL:  "https://[2001:db8::1]:8443/realms/trino",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &authv1alpha1.OIDCProvider{
				Hostname:       test.hostname,
				Port:           test.port,
				PrincipalClaim: testOIDCPrincipalClaim,
				RootPath:       testOIDCRootPath,
				TLS:            test.tls,
			}

			got, err := oidcIssuer(provider)
			if err != nil {
				t.Fatalf("oidcIssuer() error = %v", err)
			}
			if got != test.wantURL {
				t.Errorf("OIDC issuer = %q, want %q", got, test.wantURL)
			}
		})
	}
}

func TestOidcIssuerRejectsMalformedHostnameAndPort(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		port     int
		want     string
	}{
		{
			name:     "scheme included",
			hostname: "https://identity.example.com",
			want:     "scheme",
		},
		{
			name:     "embedded port",
			hostname: "identity.example.com:8443",
			want:     "provider.port",
		},
		{
			name:     "mismatched IPv6 brackets",
			hostname: "[" + testOIDCIPv6Hostname,
			want:     "mismatched IPv6 brackets",
		},
		{
			name:     "bracketed DNS name",
			hostname: "[identity.example.com]",
			want:     "valid bracketed IPv6",
		},
		{
			name:     "invalid DNS label",
			hostname: "bad_host.example.com",
			want:     "valid DNS hostname",
		},
		{
			name:     "port above maximum",
			hostname: testOIDCHostname,
			port:     65536,
			want:     "between 0 and 65535",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := oidcIssuer(&authv1alpha1.OIDCProvider{
				Hostname: test.hostname,
				Port:     test.port,
			})
			if err == nil {
				t.Fatal("oidcIssuer() error = nil, want malformed issuer error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("oidcIssuer() error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestOidcConfiguresOAuth2ForWebUI(t *testing.T) {
	oidc := testOIDCAuthenticator(
		t,
		&trinov1alpha1.OidcSpec{ClientCredentialsSecret: testOIDCClientSecret},
		testOIDCProvider(),
	)

	properties := oidc.GetConfigProperties()
	if got := properties["web-ui.authentication.type"]; got != oidcAuthenticationType {
		t.Errorf("web-ui.authentication.type = %q, want %q", got, oidcAuthenticationType)
	}
}

func TestOidcScopesMergeDefaultsProviderAndClusterInFirstSeenOrder(t *testing.T) {
	provider := testOIDCProvider()
	provider.Scopes = []string{
		oidcScopeOpenID,
		testOIDCGroupsScope,
		oidcScopeEmail,
		"offline_access",
		testOIDCGroupsScope,
	}
	oidc := testOIDCAuthenticator(
		t,
		&trinov1alpha1.OidcSpec{
			ClientCredentialsSecret: testOIDCClientSecret,
			ExtraScopes: []string{
				testOIDCGroupsScope,
				"custom",
				oidcScopeProfile,
				"offline_access",
				"custom",
				"",
			},
		},
		provider,
	)

	got := oidc.GetConfigProperties()["http-server.authentication.oauth2.scopes"]
	want := "openid email profile groups offline_access custom"
	if got != want {
		t.Errorf("OIDC scopes = %q, want %q", got, want)
	}
}

func TestAuthenticatorFactoryValidatesRequiredOIDCFields(t *testing.T) {
	tests := []struct {
		name     string
		config   *trinov1alpha1.OidcSpec
		provider *authv1alpha1.OIDCProvider
		want     string
	}{
		{
			name:     "empty hostname",
			config:   &trinov1alpha1.OidcSpec{ClientCredentialsSecret: testOIDCClientSecret},
			provider: &authv1alpha1.OIDCProvider{Hostname: " ", PrincipalClaim: testOIDCPrincipalClaim},
			want:     "empty OIDC hostname",
		},
		{
			name:     "empty principal claim",
			config:   &trinov1alpha1.OidcSpec{ClientCredentialsSecret: testOIDCClientSecret},
			provider: &authv1alpha1.OIDCProvider{Hostname: testOIDCHostname, PrincipalClaim: " "},
			want:     "empty OIDC principalClaim",
		},
		{
			name:     "empty client credentials secret",
			config:   &trinov1alpha1.OidcSpec{ClientCredentialsSecret: " "},
			provider: testOIDCProvider(),
			want:     "empty OIDC clientCredentialsSecret",
		},
		{
			name:   "malformed hostname",
			config: &trinov1alpha1.OidcSpec{ClientCredentialsSecret: testOIDCClientSecret},
			provider: &authv1alpha1.OIDCProvider{
				Hostname:       "https://identity.example.com",
				PrincipalClaim: testOIDCPrincipalClaim,
			},
			want: "invalid OIDC issuer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := authenticatorFactory(
				"invalid-oidc",
				test.config,
				&authv1alpha1.AuthenticationProvider{OIDC: test.provider},
			)
			if err == nil {
				t.Fatal("authenticatorFactory() error = nil, want invalid OIDC configuration error")
			}
			if !strings.Contains(err.Error(), "invalid-oidc") || !strings.Contains(err.Error(), test.want) {
				t.Errorf("authenticatorFactory() error = %q, want class name and %q", err, test.want)
			}
		})
	}
}

func TestAuthenticatorFactoryValidatesOIDCTLSVerification(t *testing.T) {
	config := &trinov1alpha1.OidcSpec{ClientCredentialsSecret: testOIDCClientSecret}
	validProvider := &authv1alpha1.AuthenticationProvider{
		OIDC: &authv1alpha1.OIDCProvider{
			Hostname:       testOIDCHostname,
			PrincipalClaim: testOIDCPrincipalClaim,
			TLS:            testWebPKIOIDCTLS(),
		},
	}
	authType, authenticator, err := authenticatorFactory("valid-oidc", config, validProvider)
	if err != nil {
		t.Fatalf("authenticatorFactory() rejected server.webPki verification: %v", err)
	}
	if authType != AuthenticationTypeOIDC || authenticator == nil {
		t.Fatalf("authenticatorFactory() = (%q, %#v), want OIDC authenticator", authType, authenticator)
	}

	webPKIServer := &commonsv1alpha1.ServerVerification{
		CACert: &commonsv1alpha1.CACert{WebPki: &commonsv1alpha1.WebPki{}},
	}
	tests := []struct {
		name string
		tls  *authv1alpha1.OIDCTls
		want string
	}{
		{
			name: "missing verification",
			tls:  &authv1alpha1.OIDCTls{},
			want: "no verification configuration",
		},
		{
			name: "empty verification",
			tls: &authv1alpha1.OIDCTls{
				Verification: &commonsv1alpha1.TLSVerificationSpec{},
			},
			want: testOIDCWebPKIPath,
		},
		{
			name: "verification none",
			tls: &authv1alpha1.OIDCTls{
				Verification: &commonsv1alpha1.TLSVerificationSpec{
					None: &commonsv1alpha1.NoneVerification{},
				},
			},
			want: "verification.none",
		},
		{
			name: "conflicting verification modes",
			tls: &authv1alpha1.OIDCTls{
				Verification: &commonsv1alpha1.TLSVerificationSpec{
					None:   &commonsv1alpha1.NoneVerification{},
					Server: webPKIServer,
				},
			},
			want: "multiple verification modes",
		},
		{
			name: "server missing CA",
			tls: &authv1alpha1.OIDCTls{
				Verification: &commonsv1alpha1.TLSVerificationSpec{
					Server: &commonsv1alpha1.ServerVerification{},
				},
			},
			want: testOIDCWebPKIPath,
		},
		{
			name: "empty CA",
			tls: &authv1alpha1.OIDCTls{
				Verification: &commonsv1alpha1.TLSVerificationSpec{
					Server: &commonsv1alpha1.ServerVerification{
						CACert: &commonsv1alpha1.CACert{},
					},
				},
			},
			want: testOIDCWebPKIPath,
		},
		{
			name: "custom CA SecretClass",
			tls: &authv1alpha1.OIDCTls{
				Verification: &commonsv1alpha1.TLSVerificationSpec{
					Server: &commonsv1alpha1.ServerVerification{
						CACert: &commonsv1alpha1.CACert{SecretClass: "custom-oidc-ca"},
					},
				},
			},
			want: "SecretClass",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &authv1alpha1.AuthenticationProvider{
				OIDC: &authv1alpha1.OIDCProvider{
					Hostname:       testOIDCHostname,
					PrincipalClaim: testOIDCPrincipalClaim,
					TLS:            test.tls,
				},
			}
			_, _, err := authenticatorFactory("invalid-oidc", config, provider)
			if err == nil {
				t.Fatal("authenticatorFactory() error = nil, want unsupported TLS verification error")
			}
			if !strings.Contains(err.Error(), "invalid-oidc") || !strings.Contains(err.Error(), test.want) {
				t.Errorf("authenticatorFactory() error = %q, want class name and %q", err, test.want)
			}
		})
	}
}

func testOIDCProvider() *authv1alpha1.OIDCProvider {
	return &authv1alpha1.OIDCProvider{
		Hostname:       testOIDCHostname,
		PrincipalClaim: testOIDCPrincipalClaim,
	}
}

func testOIDCAuthenticator(
	t *testing.T,
	config *trinov1alpha1.OidcSpec,
	provider *authv1alpha1.OIDCProvider,
) *Oidc {
	t.Helper()

	issuer, err := oidcIssuer(provider)
	if err != nil {
		t.Fatalf("build test OIDC issuer: %v", err)
	}
	return &Oidc{
		Config:   config,
		Provider: provider,
		issuer:   issuer,
	}
}

func testWebPKIOIDCTLS() *authv1alpha1.OIDCTls {
	return &authv1alpha1.OIDCTls{
		Verification: &commonsv1alpha1.TLSVerificationSpec{
			Server: &commonsv1alpha1.ServerVerification{
				CACert: &commonsv1alpha1.CACert{
					WebPki: &commonsv1alpha1.WebPki{},
				},
			},
		},
	}
}
