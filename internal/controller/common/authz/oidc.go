package authz

import (
	"net/url"
	"strconv"
	"strings"

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

// func (o *Oidc) getEnvNamePrefix() string {

// 	hash := sha256.New()
// 	hash.Write([]byte(o.Config.ClientCredentialsSecret))
// 	hashedSecretName := hex.EncodeToString(hash.Sum(nil))

// 	return fmt.Sprintf("OIDC_%s", hashedSecretName)
// }

func (o *Oidc) GetConfigProperties() *properties.Properties {
	scopes := make([]string, 3, 3+len(o.Config.ExtraScopes))
	scopes[0] = "openid"
	scopes[1] = "email"
	scopes[2] = "profile"
	issuer := url.URL{
		Scheme: "http",
		Host:   o.Provider.Hostname,
		Path:   o.Provider.RootPath,
	}

	if (issuer.Scheme == "http" && o.Provider.Port != 80) || (issuer.Scheme == "https" && o.Provider.Port != 443) {
		issuer.Host += ":" + strconv.Itoa(o.Provider.Port)
	}

	scopes = append(scopes, o.Config.ExtraScopes...)

	p := properties.NewProperties()
	p.Add("http-server.authentication.type", "OAUTH2")
	p.Add("http-server.authentication.oauth2.scopes", strings.Join(scopes, " "))
	p.Add("http-server.authentication.oauth2.client-id", "${ENV:OIDC_CLIENT_ID}")
	p.Add("http-server.authentication.oauth2.client-secret", "${ENV:OIDC_CLIENT_SECRET}")
	p.Add("http-server.authentication.oauth2.issuer", issuer.String())
	p.Add("http-server.authentication.oauth2.principal-field", o.Provider.PrincipalClaim)

	return p
}

func (o *Oidc) GetEnvVars() []corev1.EnvVar {
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
