package authz

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/config/properties"
	corev1 "k8s.io/api/core/v1"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
)

var _ Authenticator = &Oidc{}

type Oidc struct {
	AuthenticationClassName string
	Config                  *trinov1alpha1.OidcSpec
	Provider                *authv1alpha1.OIDCProvider
}

func (o *Oidc) getEnvNamePrefix() string {

	hash := sha256.New()
	hash.Write([]byte(o.Config.ClientCredentialsSecret))
	hashedSecretName := hex.EncodeToString(hash.Sum(nil))

	return fmt.Sprintf("OIDC_%s", hashedSecretName)
}

func (o *Oidc) GetConfigProperties() *properties.Properties {
	scopes := []string{"openid", "email", "profile"}
	issuer := url.URL{
		Scheme: "http",
		Host:   o.Provider.Hostname,
		Path:   o.Provider.RootPath,
	}

	if o.Provider.Port != 0 {
		issuer.Host += ":" + strconv.Itoa(o.Provider.Port)
	}

	scopes = append(scopes, o.Config.ExtraScopes...)

	p := properties.NewProperties()
	p.Add("http-server.authentication.type", "OAUTH2")
	p.Add("http-server.authentication.oauth2.client-id", fmt.Sprintf("${ENV:%s_CLIENT_ID}", o.getEnvNamePrefix()))
	p.Add("http-server.authentication.oauth2.client-secret", fmt.Sprintf("${ENV:%s_CLIENT_SECRET}", o.getEnvNamePrefix()))
	p.Add("http-server.authentication.oauth2.issuer", issuer.String())
	p.Add("http-server.authentication.oauth2.principal-field", o.Provider.PrincipalClaim)

	return p
}

func (o *Oidc) GetEnvVars() []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name: fmt.Sprintf("%s_CLIENT_ID", o.getEnvNamePrefix()),
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
			Name: fmt.Sprintf("%s_CLIENT_SECRET", o.getEnvNamePrefix()),
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
