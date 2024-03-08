package controller

import (
	"context"
	"fmt"
	trinov1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	"github.com/zncdata-labs/trino-operator/internal/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type ClusterConfigMapReconciler struct {
	common.MultiConfigurationStyleReconciler[*trinov1alpha1.TrinoCluster, *trinov1alpha1.RoleGroupSpec]
}

// NewClusterConfigMap new a ClusterConfigMapReconcile
func NewClusterConfigMap(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	labels map[string]string,
	mergedCfg *trinov1alpha1.RoleGroupSpec,

) *ClusterConfigMapReconciler {
	return &ClusterConfigMapReconciler{
		MultiConfigurationStyleReconciler: *common.NewMultiConfigurationStyleReconciler(
			scheme,
			instance,
			client,
			groupName,
			labels,
			mergedCfg,
		),
	}
}

// Build implements the MultiResourceReconcilerBuilder interface
func (c *ClusterConfigMapReconciler) Build(_ context.Context) ([]common.ResourceBuilder, error) {
	return []common.ResourceBuilder{
		c.createCatalogConfigmapReconciler(),
		c.createSchemaConfigmapReconciler(),
	}, nil
}

// create catalog configmap reconciler
func (c *ClusterConfigMapReconciler) createCatalogConfigmapReconciler() common.ResourceBuilder {
	return NewGeneralConfigMap(
		c.Scheme,
		c.Instance,
		c.Client,
		c.GroupName,
		c.MergedLabels,
		c.MergedCfg,
		c.createCatalogConfigmap)
}

// create schema configmap reconciler
func (c *ClusterConfigMapReconciler) createSchemaConfigmapReconciler() common.ResourceBuilder {
	return NewGeneralConfigMap(
		c.Scheme,
		c.Instance,
		c.Client,
		c.GroupName,
		c.MergedLabels,
		c.MergedCfg,
		c.createSchemaConfigmap)
}

// create catalog configmap resource
const tpchProps = `connector.name=tpch
tpch.splits-per-node=4
`

const tpcdsProps = `connector.name=tpcds
tpcds.splits-per-node=4
`

func (c *ClusterConfigMapReconciler) createCatalogConfigmap() client.Object {
	labels := c.Instance.GetLabels()

	additionalCatalogs := make(map[string]string)
	for catalogName, catalogProperties := range c.Instance.Spec.ClusterConfig.Catalogs {
		key := fmt.Sprintf("%s.properties", catalogName)
		additionalCatalogs[key] = fmt.Sprintf("%s\n", indentProperties(catalogProperties))
	}
	data := map[string]string{
		"tpch.properties":  tpchProps,
		"tpcds.properties": tpcdsProps,
	}
	for key, value := range additionalCatalogs {
		data[key] = value
	}
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.CreateCatalogConfigmapName(c.Instance.Name),
			Namespace: c.Instance.Namespace,
			Labels:    labels,
		},
		Data: data,
	}
	return &cm
}

// create schema configmap resource
func (c *ClusterConfigMapReconciler) createSchemaConfigmap() client.Object {
	labels := c.Instance.GetLabels()
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.CreateSchemaConfigmapName(c.Instance.Name),
			Namespace: c.Instance.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	}
	return &cm
}

// trim space of each line
func indentProperties(catalogContent string) string {
	var res string
	for _, line := range strings.Split(catalogContent, "\n") {
		res = res + strings.TrimSpace(line) + "\n"
	}
	return res
}

// GeneralConfigMapReconciler general config map reconciler generator
// it can be used to generate config map reconciler for simple config map
// such as trino catalog, schema config map
// parameters:
// 1. resourceBuilerFunc: a function to create a new resource
type GeneralConfigMapReconciler struct {
	common.GeneralResourceStyleReconciler[*trinov1alpha1.TrinoCluster, *trinov1alpha1.RoleGroupSpec]
	resourceBuilderFunc func() client.Object
}

// NewGeneralConfigMap new a GeneralConfigMapReconciler
func NewGeneralConfigMap(
	scheme *runtime.Scheme,
	instance *trinov1alpha1.TrinoCluster,
	client client.Client,
	groupName string,
	mergedLabels map[string]string,
	mergedCfg *trinov1alpha1.RoleGroupSpec,
	resourceBuilderFunc func() client.Object,

) *GeneralConfigMapReconciler {
	return &GeneralConfigMapReconciler{
		GeneralResourceStyleReconciler: *common.NewGeneraResourceStyleReconciler(
			scheme,
			instance,
			client,
			groupName,
			mergedLabels,
			mergedCfg,
		),
		resourceBuilderFunc: resourceBuilderFunc,
	}

}

// Build implements the ResourceBuilder interface
func (c *GeneralConfigMapReconciler) Build(_ context.Context) (client.Object, error) {
	return c.resourceBuilderFunc(), nil
}
