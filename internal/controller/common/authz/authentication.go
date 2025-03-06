package authz

import (
	"context"
	"fmt"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/config/properties"
	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

type AuthenticationType string

const (
	AuthenticationTypeOIDC   AuthenticationType = "oidc"
	AuthenticationTypeLDAP   AuthenticationType = "ldap"
	AuthenticationTypeStatic AuthenticationType = "static"
	AuthenticationTypeTls    AuthenticationType = "tls"
)

type Authenticator interface {
	GetEnvVars() []corev1.EnvVar
	GetVolumes() []corev1.Volume
	GetVolumeMounts() []corev1.VolumeMount
	GetConfigProperties() *properties.Properties
	GetCommands() []string
}

func AuthenticatorFectory(config *trinov1alpha1.OidcSpec, provider *authv1alpha1.AuthenticationProvider) (AuthenticationType, Authenticator) {
	if provider.OIDC != nil {
		return AuthenticationTypeOIDC, &Oidc{Config: config, Provider: provider.OIDC}
	} else if provider.Static != nil {
		return AuthenticationTypeStatic, &Static{Provider: provider.Static}
	} else if provider.LDAP != nil {
		return AuthenticationTypeLDAP, &Ldap{Provider: provider.LDAP}
	} else {
		return "", nil
	}
}

var _ Authenticator = &TrinoAuthentication{}

type TrinoAuthentication struct {
	Authenticators []Authenticator
}

func NewAuthentication(
	ctx context.Context,
	client *client.Client,
	authentication []trinov1alpha1.AuthenticationSpec,
) (*TrinoAuthentication, error) {
	authenticators := make(map[AuthenticationType]Authenticator)

	for _, authenticationSpec := range authentication {
		name := authenticationSpec.AuthenticationClass
		obj := &authv1alpha1.AuthenticationClass{}
		if err := client.Client.Get(
			ctx,
			ctrlclient.ObjectKey{Namespace: client.GetOwnerNamespace(), Name: name},
			obj,
		); ctrlclient.IgnoreNotFound(err) != nil {
			return nil, err
		}

		authType, authenticator := AuthenticatorFectory(authenticationSpec.Oidc, obj.Spec.AuthenticationProvider)

		if authenticator != nil {
			if _, ok := authenticators[authType]; ok {
				return nil, fmt.Errorf("Can not support multiple authenticators of the same type. Found multiple %s authenticators in AuthenticationClass %s", authType, name)
			}
			authenticators[authType] = authenticator
		}
	}

	authenticatorList := make([]Authenticator, 0, len(authenticators))
	for _, authenticator := range authenticators {
		authenticatorList = append(authenticatorList, authenticator)
	}
	return &TrinoAuthentication{Authenticators: authenticatorList}, nil
}

func (a *TrinoAuthentication) GetEnvVars() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	for _, authenticator := range a.Authenticators {
		envVars = append(envVars, authenticator.GetEnvVars()...)
	}

	return envVars
}

func (a *TrinoAuthentication) GetVolumes() []corev1.Volume {
	volumes := []corev1.Volume{}

	for _, authenticator := range a.Authenticators {
		volumes = append(volumes, authenticator.GetVolumes()...)
	}

	return volumes
}

func (a *TrinoAuthentication) GetVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}

	for _, authenticator := range a.Authenticators {
		volumeMounts = append(volumeMounts, authenticator.GetVolumeMounts()...)
	}

	return volumeMounts
}

func (a *TrinoAuthentication) GetConfigProperties() *properties.Properties {
	p := properties.NewProperties()

	for _, authenticator := range a.Authenticators {
		for _, key := range authenticator.GetConfigProperties().Keys() {
			value, _ := authenticator.GetConfigProperties().Get(key)
			p.Add(key, value)
		}
	}

	return p
}

func (a *TrinoAuthentication) GetCommands() []string {
	commands := []string{}

	for _, authenticator := range a.Authenticators {
		commands = append(commands, authenticator.GetCommands()...)
	}

	return commands
}
