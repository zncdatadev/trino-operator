package authz

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

const (
	oidcSchemeHTTP  = "http"
	oidcSchemeHTTPS = "https"

	oidcDefaultHTTPPort  = 80
	oidcDefaultHTTPSPort = 443

	oidcScopeOpenID  = "openid"
	oidcScopeEmail   = "email"
	oidcScopeProfile = "profile"

	oidcAuthenticationType = "OAUTH2"
)

var _ Authenticator = &Oidc{}

type Oidc struct {
	AuthenticationClassName string
	Config                  *trinov1alpha1.OidcSpec
	Provider                *authv1alpha1.OIDCProvider
	issuer                  string
}

func (o *Oidc) GetConfigProperties() map[string]string {
	if o == nil || o.Config == nil || o.Provider == nil {
		return map[string]string{}
	}

	return map[string]string{
		"http-server.authentication.type":                   oidcAuthenticationType,
		"web-ui.authentication.type":                        oidcAuthenticationType,
		"http-server.authentication.oauth2.scopes":          strings.Join(mergeOIDCScopes(o.Provider.Scopes, o.Config.ExtraScopes), " "),
		"http-server.authentication.oauth2.client-id":       "${ENV:OIDC_CLIENT_ID}",
		"http-server.authentication.oauth2.client-secret":   "${ENV:OIDC_CLIENT_SECRET}",
		"http-server.authentication.oauth2.issuer":          o.issuer,
		"http-server.authentication.oauth2.principal-field": o.Provider.PrincipalClaim,
	}
}

func oidcIssuer(provider *authv1alpha1.OIDCProvider) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("OIDC provider must not be nil")
	}

	hostname, isIPv6, err := normalizeOIDCHostname(provider.Hostname)
	if err != nil {
		return "", err
	}
	if provider.Port < 0 || provider.Port > 65535 {
		return "", fmt.Errorf("OIDC port %d must be between 0 and 65535", provider.Port)
	}

	scheme := oidcSchemeHTTP
	defaultPort := oidcDefaultHTTPPort
	if provider.TLS != nil {
		scheme = oidcSchemeHTTPS
		defaultPort = oidcDefaultHTTPSPort
	}

	host := hostname
	if provider.Port != 0 && provider.Port != defaultPort {
		host = net.JoinHostPort(hostname, strconv.Itoa(provider.Port))
	} else if isIPv6 {
		host = "[" + hostname + "]"
	}

	return (&url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   provider.RootPath,
	}).String(), nil
}

func normalizeOIDCHostname(rawHostname string) (string, bool, error) {
	hostname := strings.TrimSpace(rawHostname)
	if hostname == "" {
		return "", false, fmt.Errorf("OIDC hostname must not be empty")
	}
	if hostname != rawHostname {
		return "", false, fmt.Errorf("OIDC hostname %q must not contain leading or trailing whitespace", rawHostname)
	}
	if strings.ContainsAny(hostname, `\/?#@%`) {
		return "", false, fmt.Errorf("OIDC hostname %q must not contain a scheme, path, query, fragment, user information, or zone", hostname)
	}

	startsWithBracket := strings.HasPrefix(hostname, "[")
	endsWithBracket := strings.HasSuffix(hostname, "]")
	if startsWithBracket || endsWithBracket {
		if !startsWithBracket || !endsWithBracket {
			return "", false, fmt.Errorf("OIDC hostname %q has mismatched IPv6 brackets", hostname)
		}
		unbracketed := strings.TrimSuffix(strings.TrimPrefix(hostname, "["), "]")
		ip := net.ParseIP(unbracketed)
		if ip == nil || ip.To4() != nil {
			return "", false, fmt.Errorf("OIDC hostname %q is not a valid bracketed IPv6 address", hostname)
		}
		return unbracketed, true, nil
	}

	if ip := net.ParseIP(hostname); ip != nil {
		return hostname, ip.To4() == nil, nil
	}
	if strings.Contains(hostname, ":") {
		return "", false, fmt.Errorf(
			"OIDC hostname %q is not a valid IPv6 address and must not include a port; configure provider.port separately",
			hostname,
		)
	}
	if !isValidDNSHostname(hostname) {
		return "", false, fmt.Errorf("OIDC hostname %q is not a valid DNS hostname or IP address", hostname)
	}
	return hostname, false, nil
}

func isValidDNSHostname(hostname string) bool {
	if len(hostname) > 253 {
		return false
	}
	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" {
		return false
	}

	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		for index := range len(label) {
			character := label[index]
			alphaNumeric := character >= 'a' && character <= 'z' ||
				character >= 'A' && character <= 'Z' ||
				character >= '0' && character <= '9'
			if !alphaNumeric && (character != '-' || index == 0 || index == len(label)-1) {
				return false
			}
		}
	}
	return true
}

func mergeOIDCScopes(providerScopes, extraScopes []string) []string {
	result := make([]string, 0, 3+len(providerScopes)+len(extraScopes))
	seen := make(map[string]struct{}, cap(result))

	appendScope := func(scope string) {
		if scope == "" {
			return
		}
		if _, exists := seen[scope]; exists {
			return
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}

	for _, scope := range []string{oidcScopeOpenID, oidcScopeEmail, oidcScopeProfile} {
		appendScope(scope)
	}
	for _, scope := range providerScopes {
		appendScope(scope)
	}
	for _, scope := range extraScopes {
		appendScope(scope)
	}

	return result
}

func (o *Oidc) GetEnvVars() []corev1.EnvVar {
	if o == nil || o.Config == nil {
		return nil
	}

	envVars := []corev1.EnvVar{
		{
			Name: "OIDC_CLIENT_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "CLIENT_ID",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: o.Config.ClientCredentialsSecret,
					},
				},
			},
		},
		{
			Name: "OIDC_CLIENT_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "CLIENT_SECRET",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: o.Config.ClientCredentialsSecret,
					},
				},
			},
		},
	}
	return envVars
}

func (o *Oidc) GetCommands() []string {
	return nil
}

func (o *Oidc) GetVolumes() []corev1.Volume {
	return nil
}

func (o *Oidc) GetVolumeMounts() []corev1.VolumeMount {
	return nil
}
