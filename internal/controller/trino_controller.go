/*
Copyright 2023 zncdata-labs.

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

	"github.com/zncdata-labs/operator-go/pkg/status"
	"github.com/zncdata-labs/operator-go/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	trinov1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TrinoReconciler reconciles a Trino object
type TrinoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinoclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinoclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=trino.zncdata.dev,resources=trinoclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Trino object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TrinoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.Log.Info("Reconciling instance")

	trino := &trinov1alpha1.TrinoCluster{}

	if err := r.Get(ctx, req.NamespacedName, trino); err != nil {
		if client.IgnoreNotFound(err) != nil {
			r.Log.Error(err, "unable to fetch Trino")
			return ctrl.Result{}, err
		}
		r.Log.Info("Trino resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	// Get the status condition, if it exists and its generation is not the
	//same as the Trino's generation, reset the status conditions
	readCondition := apimeta.FindStatusCondition(trino.Status.Conditions, trinov1alpha1.ConditionTypeProgressing)
	if readCondition == nil || readCondition.ObservedGeneration != trino.GetGeneration() {
		trino.InitStatusConditions()

		if err := r.UpdateStatus(ctx, trino); err != nil {
			r.Log.Error(err, "unable to update status for Trino")
			return ctrl.Result{}, err
		}
	}

	r.Log.Info("Trino found", "Name", trino.Name)

	if err := r.reconcileDeployment(ctx, trino); err != nil {
		r.Log.Error(err, "unable to reconcile Deployment")
		return ctrl.Result{}, err
	}

	if updated := trino.Status.SetStatusCondition(metav1.Condition{
		Type:               status.ConditionTypeReconcileDeployment,
		Status:             metav1.ConditionTrue,
		Reason:             status.ConditionReasonRunning,
		Message:            "TrinoServer's deployment is running",
		ObservedGeneration: trino.GetGeneration(),
	}); updated {
		err := utils.UpdateStatus(ctx, r.Client, trino)
		if err != nil {
			r.Log.Error(err, "unable to update status for Deployment")
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileService(ctx, trino); err != nil {
		r.Log.Error(err, "unable to reconcile Service")
		return ctrl.Result{}, err
	}

	if updated := trino.Status.SetStatusCondition(metav1.Condition{
		Type:               status.ConditionTypeReconcileService,
		Status:             metav1.ConditionTrue,
		Reason:             status.ConditionReasonRunning,
		Message:            "TrinoServer's service is running",
		ObservedGeneration: trino.GetGeneration(),
	}); updated {
		err := utils.UpdateStatus(ctx, r.Client, trino)
		if err != nil {
			r.Log.Error(err, "unable to update status for Service")
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileIngress(ctx, trino); err != nil {
		r.Log.Error(err, "unable to reconcile Ingress")
		return ctrl.Result{}, err
	}
	if updated := trino.Status.SetStatusCondition(metav1.Condition{
		Type:               status.ConditionTypeReconcileIngress,
		Status:             metav1.ConditionTrue,
		Reason:             status.ConditionReasonRunning,
		Message:            "TrinoServer's ingress is running",
		ObservedGeneration: trino.GetGeneration(),
	}); updated {
		err := utils.UpdateStatus(ctx, r.Client, trino)
		if err != nil {
			r.Log.Error(err, "unable to update status for Ingress")
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileConfigMap(ctx, trino); err != nil {
		r.Log.Error(err, "unable to reconcile Ingress")
		return ctrl.Result{}, err
	}

	if !trino.Status.IsAvailable() {

		trino.SetStatusCondition(metav1.Condition{
			Type:               trinov1alpha1.ConditionTypeAvailable,
			Status:             metav1.ConditionTrue,
			Reason:             trinov1alpha1.ConditionReasonRunning,
			Message:            "Trino is running",
			ObservedGeneration: trino.GetGeneration(),
		})
		if err := r.UpdateStatus(ctx, trino); err != nil {
			r.Log.Error(err, "unable to update status for Trino")
			return ctrl.Result{}, err
		}

	}

	r.Log.Info("Successfully reconciled Trino")
	return ctrl.Result{}, nil
}

// UpdateStatus updates the status of the Trino resource
// https://stackoverflow.com/questions/76388004/k8s-controller-update-status-and-condition
func (r *TrinoReconciler) UpdateStatus(ctx context.Context, instance *trinov1alpha1.TrinoCluster) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.Status().Update(ctx, instance)
		//return r.Status().Patch(ctx, instance, client.MergeFrom(instance))
	})

	if retryErr != nil {
		r.Log.Error(retryErr, "Failed to update vfm status after retries")
		return retryErr
	}

	r.Log.V(1).Info("Successfully patched object status")
	return nil
}

func (r *TrinoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trinov1alpha1.TrinoCluster{}).
		Complete(r)
}
