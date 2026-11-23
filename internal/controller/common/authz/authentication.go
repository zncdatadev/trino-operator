package authz

import (
	"context"
	"fmt"
	"strings"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

type AuthenticationType string

const (
	AuthenticationTypeOIDC     AuthenticationType = "oidc"
	AuthenticationTypeLDAP     AuthenticationType = "ldap"
	AuthenticationTypeStatic   AuthenticationType = "static"
	AuthenticationTypeTls      AuthenticationType = "tls"
	authenticationTypeProperty                    = "http-server.authentication.type"
)

type Authenticator interface {
	GetEnvVars() []corev1.EnvVar
	GetVolumes() []corev1.Volume
	GetVolumeMounts() []corev1.VolumeMount
	GetConfigProperties() map[string]string
	GetCommands() []string
}

func authenticatorFactory(
	authenticationClassName string,
	config *trinov1alpha1.OidcSpec,
	provider *authv1alpha1.AuthenticationProvider,
) (AuthenticationType, Authenticator, error) {
	configuredProviders := configuredAuthenticationProviders(provider)
	switch len(configuredProviders) {
	case 0:
		return "", nil, fmt.Errorf(
			"authentication class %q has no provider configured; exactly one provider is required",
			authenticationClassName,
		)
	case 1:
		// Continue below so the selected provider can be validated and constructed.
	default:
		return "", nil, fmt.Errorf(
			"authentication class %q configures multiple providers (%s); exactly one provider is required",
			authenticationClassName,
			strings.Join(configuredProviders, ", "),
		)
	}

	if provider.OIDC != nil {
		issuer, err := validateOIDC(authenticationClassName, config, provider.OIDC)
		if err != nil {
			return "", nil, err
		}
		return AuthenticationTypeOIDC, &Oidc{
			AuthenticationClassName: authenticationClassName,
			Config:                  config,
			Provider:                provider.OIDC,
			issuer:                  issuer,
		}, nil
	}
	if provider.Static != nil {
		return "", nil, fmt.Errorf(
			"authentication class %q uses static authentication, which is not supported by this Trino operator",
			authenticationClassName,
		)
	}
	if provider.LDAP != nil {
		return "", nil, fmt.Errorf(
			"authentication class %q uses LDAP authentication, which is not supported by this Trino operator",
			authenticationClassName,
		)
	}

	return "", nil, fmt.Errorf("authentication class %q uses an unsupported provider", authenticationClassName)
}

func configuredAuthenticationProviders(provider *authv1alpha1.AuthenticationProvider) []string {
	if provider == nil {
		return nil
	}

	configuredProviders := make([]string, 0, 5)
	if provider.OIDC != nil {
		configuredProviders = append(configuredProviders, string(AuthenticationTypeOIDC))
	}
	if provider.TLS != nil {
		configuredProviders = append(configuredProviders, string(AuthenticationTypeTls))
	}
	if provider.Static != nil {
		configuredProviders = append(configuredProviders, string(AuthenticationTypeStatic))
	}
	if provider.LDAP != nil {
		configuredProviders = append(configuredProviders, string(AuthenticationTypeLDAP))
	}
	if provider.Kerberos != nil {
		configuredProviders = append(configuredProviders, "kerberos")
	}
	return configuredProviders
}

func validateOIDC(
	authenticationClassName string,
	config *trinov1alpha1.OidcSpec,
	provider *authv1alpha1.OIDCProvider,
) (string, error) {
	if config == nil {
		return "", fmt.Errorf(
			"authentication class %q uses OIDC but no OIDC configuration was provided",
			authenticationClassName,
		)
	}
	if strings.TrimSpace(provider.Hostname) == "" {
		return "", fmt.Errorf("authentication class %q has an empty OIDC hostname", authenticationClassName)
	}
	if strings.TrimSpace(provider.PrincipalClaim) == "" {
		return "", fmt.Errorf("authentication class %q has an empty OIDC principalClaim", authenticationClassName)
	}
	if strings.TrimSpace(config.ClientCredentialsSecret) == "" {
		return "", fmt.Errorf("authentication class %q has an empty OIDC clientCredentialsSecret", authenticationClassName)
	}
	if err := validateOIDCTLS(authenticationClassName, provider.TLS); err != nil {
		return "", err
	}

	issuer, err := oidcIssuer(provider)
	if err != nil {
		return "", fmt.Errorf("authentication class %q has an invalid OIDC issuer: %w", authenticationClassName, err)
	}
	return issuer, nil
}

func validateOIDCTLS(authenticationClassName string, tls *authv1alpha1.OIDCTls) error {
	if tls == nil {
		return nil
	}
	verification := tls.Verification
	if verification == nil {
		return fmt.Errorf(
			"authentication class %q enables OIDC TLS but has no verification configuration; configure server.caCert.webPki",
			authenticationClassName,
		)
	}
	if verification.None != nil && verification.Server != nil {
		return fmt.Errorf(
			"authentication class %q configures multiple verification modes for OIDC TLS; use exactly server.caCert.webPki",
			authenticationClassName,
		)
	}
	if verification.None != nil {
		return fmt.Errorf(
			"authentication class %q uses OIDC TLS verification.none, which is not supported; configure server.caCert.webPki",
			authenticationClassName,
		)
	}
	if verification.Server == nil || verification.Server.CACert == nil {
		return fmt.Errorf(
			"authentication class %q must configure OIDC TLS verification.server.caCert.webPki",
			authenticationClassName,
		)
	}

	caCert := verification.Server.CACert
	if caCert.SecretClass != "" {
		return fmt.Errorf(
			"authentication class %q uses custom OIDC CA SecretClass %q, which is not supported; configure server.caCert.webPki",
			authenticationClassName,
			caCert.SecretClass,
		)
	}
	if caCert.WebPki == nil {
		return fmt.Errorf(
			"authentication class %q must configure OIDC TLS verification.server.caCert.webPki",
			authenticationClassName,
		)
	}
	return nil
}

var _ Authenticator = &TrinoAuthentication{}

type TrinoAuthentication struct {
	Authenticators []Authenticator
}

func NewAuthentication(
	ctx context.Context,
	k8sClient ctrlclient.Client,
	namespace string,
	authentication []trinov1alpha1.AuthenticationSpec,
) (*TrinoAuthentication, error) {
	authenticators := make([]Authenticator, 0, len(authentication))
	seenTypes := make(map[AuthenticationType]string, len(authentication))

	for _, authenticationSpec := range authentication {
		name := authenticationSpec.AuthenticationClass
		obj := &authv1alpha1.AuthenticationClass{}
		// AuthenticationClass is cluster-scoped. namespace identifies the TrinoCluster that
		// contains the reference and is retained in the error to make a bad reference actionable.
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, obj); err != nil {
			return nil, fmt.Errorf("get authentication class %q referenced from namespace %q: %w", name, namespace, err)
		}

		authType, authenticator, err := authenticatorFactory(name, authenticationSpec.Oidc, obj.Spec.AuthenticationProvider)
		if err != nil {
			return nil, err
		}

		if previousClass, exists := seenTypes[authType]; exists {
			return nil, fmt.Errorf(
				"cannot configure multiple %s authenticators: authentication classes %q and %q use the same type",
				authType,
				previousClass,
				name,
			)
		}
		seenTypes[authType] = name
		authenticators = append(authenticators, authenticator)
	}

	return &TrinoAuthentication{Authenticators: authenticators}, nil
}

func (a *TrinoAuthentication) GetEnvVars() []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0, len(a.Authenticators))

	for _, authenticator := range a.Authenticators {
		envVars = append(envVars, authenticator.GetEnvVars()...)
	}

	return envVars
}

func (a *TrinoAuthentication) GetVolumes() []corev1.Volume {
	volumes := make([]corev1.Volume, 0, len(a.Authenticators))

	for _, authenticator := range a.Authenticators {
		volumes = append(volumes, authenticator.GetVolumes()...)
	}

	return volumes
}

func (a *TrinoAuthentication) GetVolumeMounts() []corev1.VolumeMount {
	volumeMounts := make([]corev1.VolumeMount, 0, len(a.Authenticators))

	for _, authenticator := range a.Authenticators {
		volumeMounts = append(volumeMounts, authenticator.GetVolumeMounts()...)
	}

	return volumeMounts
}

func (a *TrinoAuthentication) GetConfigProperties() map[string]string {
	result := make(map[string]string)
	authenticationTypes := make([]string, 0, len(a.Authenticators))
	seenAuthenticationTypes := make(map[string]struct{}, len(a.Authenticators))

	for _, authenticator := range a.Authenticators {
		for key, value := range authenticator.GetConfigProperties() {
			if key == authenticationTypeProperty {
				for _, authenticationType := range strings.Split(value, ",") {
					authenticationType = strings.TrimSpace(authenticationType)
					if authenticationType == "" {
						continue
					}
					if _, exists := seenAuthenticationTypes[authenticationType]; exists {
						continue
					}
					seenAuthenticationTypes[authenticationType] = struct{}{}
					authenticationTypes = append(authenticationTypes, authenticationType)
				}
				continue
			}
			result[key] = value
		}
	}
	if len(authenticationTypes) > 0 {
		result[authenticationTypeProperty] = strings.Join(authenticationTypes, ",")
	}

	return result
}

func (a *TrinoAuthentication) GetCommands() []string {
	commands := make([]string, 0, len(a.Authenticators))

	for _, authenticator := range a.Authenticators {
		commands = append(commands, authenticator.GetCommands()...)
	}

	return commands
}
