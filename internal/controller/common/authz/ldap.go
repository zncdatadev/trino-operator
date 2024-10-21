package authz

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/config/properties"
	"github.com/zncdatadev/operator-go/pkg/constants"
	"github.com/zncdatadev/operator-go/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ Authenticator = &Ldap{}

type Ldap struct {
	AuthenticationClassName string
	Provider                *authv1alpha1.LDAPProvider
}

// GetCommands implements Authenticator.
func (l *Ldap) GetCommands() []string {

	userEnvName := l.getEnvName("LDAP_USER")
	passwordEnvName := l.getEnvName("LDAP_PASSWORD")
	userFile := path.Join(l.getBindCredentialsMountPath(), "user")
	passwordFile := path.Join(l.getBindCredentialsMountPath(), "password")
	s := `
set +x

export ` + userEnvName + `=cat(` + userFile + `)
export ` + passwordEnvName + `=cat(` + passwordFile + `)
set -x
`

	return []string{util.IndentTab4Spaces(s)}
}

func (l *Ldap) getBindCredentialsMountPath() string {
	return path.Join(constants.KubedoopTlsDir, l.Provider.BindCredentials.SecretClass)
}

func (l *Ldap) getEndpoint() string {
	schema := "ldap"
	if l.Provider.TLS != nil {
		schema = "ldaps"
	}
	host := l.Provider.Hostname
	if l.Provider.Port != 0 {
		host = host + ":" + strconv.Itoa(l.Provider.Port)
	}

	return schema + "://" + host
}

func (l *Ldap) getEnvName(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(l.AuthenticationClassName, "-", "_"))
}

// GetConfigProperties implements Authenticator.
func (l *Ldap) GetConfigProperties() *properties.Properties {
	p := properties.NewProperties()

	p.Add(("http-server.authentication.type"), "PASSWORD")
	p.Add(("http-server.authentication.ldap.server"), l.getEndpoint())
	p.Add("password-authenticator.name", "ldap")
	p.Add("ldap.user-base-dn", l.Provider.SearchBase)
	p.Add("ldap.group-auth-pattern", fmt.Sprintf("(&(%s={user}))", l.Provider.LDAPFieldNames.Uid))

	// bindCredentials is required
	p.Add("ldap.bind-dn", fmt.Sprintf("${ENV:%s}", l.getEnvName("LDAP_USER")))
	p.Add("ldap.bind-password", fmt.Sprintf("${ENV:%s}", l.getEnvName("LDAP_PASSWORD")))

	// TODO: Add ldap tls support

	return p
}

// GetEnvVars implements Authenticator.
func (l *Ldap) GetEnvVars() []v1.EnvVar {
	// use secret class pass the bind credentials
	return nil
}

func (l *Ldap) getBindCredentialsVolumeName() string {
	return "ldap-bind-credentials"
}

// GetVolumeMounts implements Authenticator.
func (l *Ldap) GetVolumeMounts() []v1.VolumeMount {
	return []v1.VolumeMount{
		{
			Name:      l.getBindCredentialsVolumeName(),
			MountPath: l.getBindCredentialsMountPath(),
		},
	}
}

// GetVolumes implements Authenticator.
func (l *Ldap) GetVolumes() []v1.Volume {
	secretClass := l.Provider.BindCredentials.SecretClass

	scopes := []string{}
	if l.Provider.BindCredentials.Scope != nil {
		if l.Provider.BindCredentials.Scope.Pod {
			scopes = append(scopes, string(constants.PodScope))
		}
		if l.Provider.BindCredentials.Scope.Node {
			scopes = append(scopes, string(constants.NodeScope))
		}
		if l.Provider.BindCredentials.Scope.Services != nil {
			for _, s := range l.Provider.BindCredentials.Scope.Services {
				scopes = append(scopes, string(constants.ServiceScope)+"="+s)
			}
		}
	}

	secretVolume := corev1.Volume{
		Name: l.getBindCredentialsVolumeName(),
		VolumeSource: corev1.VolumeSource{
			Ephemeral: &corev1.EphemeralVolumeSource{
				VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							constants.AnnotationSecretsClass: secretClass,
							constants.AnnotationSecretsScope: strings.Join(scopes, constants.CommonDelimiter),
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
						StorageClassName: constants.SecretStorageClassPtr(),
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Mi"),
							},
						},
					},
				},
			},
		},
	}

	return []v1.Volume{secretVolume}
}
