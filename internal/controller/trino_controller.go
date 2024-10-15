/*
Copyright 2023 zncdatadev.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/zncdatadev/operator-go/pkg/client"
	"github.com/zncdatadev/operator-go/pkg/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	trinov1alpha1 "github.com/zncdatadev/trino-operator/api/v1alpha1"
	"github.com/zncdatadev/trino-operator/internal/controller/cluster"
)

// TrinoReconciler reconciles a TrinoCluster object
type TrinoReconciler struct {
	ctrlclient.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinocatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinoclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinoclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinoclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TrinoCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TrinoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.Log.Info("Reconciling instance")

	instance := &trinov1alpha1.TrinoCluster{}

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if ctrlclient.IgnoreNotFound(err) != nil {
			r.Log.Error(err, "unable to fetch TrinoCluster")
			return ctrl.Result{}, err
		}
		r.Log.Info("TrinoCluster resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	r.Log.Info("TrinoCluster found", "Name", instance.Name)

	resourceClient := &client.Client{Client: r.Client, OwnerReference: instance}
	gvk := instance.GetObjectKind().GroupVersionKind()

	clusterReconcoler := cluster.NewClusterReconciler(
		resourceClient,
		reconciler.ClusterInfo{
			GVK: &metav1.GroupVersionKind{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    gvk.Kind,
			},
			ClusterName: instance.Name,
		},
		&instance.Spec,
	)

	if err := clusterReconcoler.RegisterResources(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return clusterReconcoler.Run(ctx)
}

func (r *TrinoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trinov1alpha1.TrinoCluster{}).
		Complete(r)
}
