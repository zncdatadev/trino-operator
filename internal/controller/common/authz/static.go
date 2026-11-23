package authz

import (
	"path"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/constant"
	corev1 "k8s.io/api/core/v1"
)

var _ Authenticator = &Static{}

type Static struct {
	AuthenticationClassName string
	Provider                *authv1alpha1.StaticProvider
}

func (s *Static) GetCommands() []string {
	return nil
}

func (s *Static) GetConfigProperties() map[string]string {
	// The mounted Secret contains plain key/value credentials and is not a Trino password file.
	// Enabling PASSWORD here without generating password-authenticator.properties and a supported
	// hashed password file would make Trino fail during startup.
	return map[string]string{}
}

func (s *Static) GetEnvVars() []corev1.EnvVar {
	return nil
}

func (s *Static) getCredentialsMountPath() string {
	return path.Join(constant.KubedoopRoot, "auth-secrets")
}

func (s *Static) getVolumeName() string {
	return "auth-secrets-" + s.AuthenticationClassName
}

func (s *Static) GetVolumeMounts() []corev1.VolumeMount {
	if s == nil || s.Provider == nil || s.Provider.UserCredentialsSecret == nil {
		return nil
	}

	return []corev1.VolumeMount{
		{
			Name:      s.getVolumeName(),
			MountPath: s.getCredentialsMountPath(),
		},
	}
}

func (s *Static) GetVolumes() []corev1.Volume {
	if s == nil || s.Provider == nil || s.Provider.UserCredentialsSecret == nil {
		return nil
	}

	return []corev1.Volume{
		{
			Name: s.getVolumeName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: s.Provider.UserCredentialsSecret.Name,
				},
			},
		},
	}
}
