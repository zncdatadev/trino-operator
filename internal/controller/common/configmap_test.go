package common_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func f(t *testing.T, s string) {
	quantity := resource.MustParse(s)
	t.Logf("quantity: %s, value: %d, decimals: %f", quantity.String(), quantity.Value(), quantity.ToDec().AsApproximateFloat64())

	// value := float64(quantity.Value()) / (1 << 30)
	// t.Logf("value: %.2f", float64(value))

	quantity = *resource.NewQuantity(quantity.ScaledValue(resource.Giga), resource.BinarySI)
	t.Logf("quantity: %s", quantity.String())
}
func TestResource(t *testing.T) {
	f(t, "1Gi")
}
