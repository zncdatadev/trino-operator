package authz

import (
	"path"

	authv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"github.com/zncdatadev/operator-go/pkg/config/properties"
	"github.com/zncdatadev/operator-go/pkg/constants"
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

func (s *Static) GetConfigProperties() *properties.Properties {
	return nil
}

func (s *Static) GetEnvVars() []corev1.EnvVar {
	return nil
}

func (s *Static) getCredentialsMountPath() string {
	return path.Join(constants.KubedoopRoot, "auth-secrets")
}

func (s *Static) getVolumeName() string {
	return "auth-secrets-" + s.AuthenticationClassName
}

func (s *Static) GetVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      s.getVolumeName(),
			MountPath: s.getCredentialsMountPath(),
		},
	}
}

func (s *Static) GetVolumes() []corev1.Volume {
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
