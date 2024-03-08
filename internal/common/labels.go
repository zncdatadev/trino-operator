package common

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type RoleLabels[T client.Object] struct {
	Cr   T
	Name string
}

func (r *RoleLabels[T]) GetLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/Name":       strings.ToLower(r.Cr.GetName()),
		"app.kubernetes.io/component":  r.Name,
		"app.kubernetes.io/managed-by": "trino-operator",
	}
}
