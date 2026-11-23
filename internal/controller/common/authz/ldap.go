package authz

import (
	"fmt"
	"path"
	"strconv"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/constant"
	"github.com/zncdatadev/operator-go/pkg/security"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ Authenticator = &Ldap{}

type Ldap struct {
	AuthenticationClassName string
	Provider                *authv1alpha1.LDAPProvider
}

// GetCommands implements Authenticator.
func (l *Ldap) GetCommands() []string {
	if l == nil || l.Provider == nil || l.Provider.BindCredentials == nil {
		return nil
	}

	userEnvName := "LDAP_USER"
	passwordEnvName := "LDAP_PASSWORD"
	userFile := path.Join(l.getBindCredentialsMountPath(), "user")
	passwordFile := path.Join(l.getBindCredentialsMountPath(), "password")
	s := `
set +x

export ` + userEnvName + `=$(cat ` + userFile + `)
export ` + passwordEnvName + `=$(cat ` + passwordFile + `)
set -x
`

	return []string{s}
}

func (l *Ldap) getBindCredentialsMountPath() string {
	return path.Join(constant.KubedoopTlsDir, l.Provider.BindCredentials.SecretClass)
}

func (l *Ldap) getEndpoint() string {
	if l == nil || l.Provider == nil {
		return ""
	}

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

// func (l *Ldap) getEnvName(prefix string) string {
// 	return fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(l.AuthenticationClassName, "-", "_"))
// }

// GetConfigProperties implements Authenticator.
func (l *Ldap) GetConfigProperties() map[string]string {
	if l == nil || l.Provider == nil {
		return map[string]string{}
	}

	uidField := "uid"
	if l.Provider.LDAPFieldNames != nil && l.Provider.LDAPFieldNames.Uid != "" {
		uidField = l.Provider.LDAPFieldNames.Uid
	}

	p := map[string]string{
		"http-server.authentication.type":        "PASSWORD",
		"http-server.authentication.ldap.server": l.getEndpoint(),
		"password-authenticator.name":            "ldap",
		"ldap.user-base-dn":                      l.Provider.SearchBase,
		"ldap.group-auth-pattern":                fmt.Sprintf("(&(%s={user}))", uidField),
	}

	if l.Provider.BindCredentials != nil {
		p["ldap.bind-dn"] = "${ENV:LDAP_USER}"
		p["ldap.bind-password"] = "${ENV:LDAP_PASSWORD}"
	}

	// TODO: Add ldap tls support

	return p
}

// GetEnvVars implements Authenticator.
func (l *Ldap) GetEnvVars() []corev1.EnvVar {
	// use secret class pass the bind credentials
	return nil
}

func (l *Ldap) getBindCredentialsVolumeName() string {
	return "ldap-bind-credentials"
}

// GetVolumeMounts implements Authenticator.
func (l *Ldap) GetVolumeMounts() []corev1.VolumeMount {
	if l == nil || l.Provider == nil || l.Provider.BindCredentials == nil {
		return nil
	}

	return []corev1.VolumeMount{
		{
			Name:      l.getBindCredentialsVolumeName(),
			MountPath: l.getBindCredentialsMountPath(),
		},
	}
}

// GetVolumes implements Authenticator.
func (l *Ldap) GetVolumes() []corev1.Volume {
	if l == nil || l.Provider == nil || l.Provider.BindCredentials == nil {
		return nil
	}

	secretClass := l.Provider.BindCredentials.SecretClass
	annotations := map[string]string{
		security.SecretClassAnnotation: secretClass,
	}
	if scope := security.ScopeString(l.Provider.BindCredentials.Scope); scope != "" {
		annotations[security.SecretClassScopeAnnotation] = scope
	}

	secretVolume := corev1.Volume{
		Name: l.getBindCredentialsVolumeName(),
		VolumeSource: corev1.VolumeSource{
			Ephemeral: &corev1.EphemeralVolumeSource{
				VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: annotations,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
						StorageClassName: ptr.To(string(security.SecretStorageClass)),
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

	return []corev1.Volume{secretVolume}
}
