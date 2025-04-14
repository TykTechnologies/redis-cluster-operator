/*
Copyright 2025.

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

package redisclusterbackup

import (
	"context"

	"github.com/go-logr/logr"
	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
	"github.com/TykTechnologies/redis-cluster-operator/internal/k8sutil"
	"github.com/TykTechnologies/redis-cluster-operator/internal/utils"
)

const backupFinalizer = "finalizer.backup.redis.kun"

// RedisClusterBackupReconciler reconciles a RedisClusterBackup object
type RedisClusterBackupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder

	crController  k8sutil.ICustomResource
	jobController k8sutil.IJobControl

	Log logr.Logger
}

// NewReconcileRedisClusterBackup creates a new instance of the reconciler.
func NewReconcileRedisClusterBackup(mgr ctrl.Manager) *RedisClusterBackupReconciler {
	// If your direct client is needed, you can create it here (or use mgr.GetClient()).
	return &RedisClusterBackupReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		recorder:      mgr.GetEventRecorderFor("redis-cluster-operator-backup"),
		crController:  k8sutil.NewCRControl(mgr.GetClient()),
		jobController: k8sutil.NewJobController(mgr.GetClient()),
		Log:           ctrl.Log.WithName("controllers").WithName("RedisClusterBackup"),
	}
}

// +kubebuilder:rbac:groups=redis.kun,resources=redisclusterbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redis.kun,resources=redisclusterbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redis.kun,resources=redisclusterbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisClusterBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *RedisClusterBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	logger.Info("Reconciling RedisClusterBackup")

	// Fetch the RedisClusterBackup instance.
	instance := &redisv1alpha1.RedisClusterBackup{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Object isn't found; could have been deleted.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// OPTIONAL: Handle finalizer logic if you need to run cleanup before deletion.
	/*
		if instance.GetDeletionTimestamp() != nil {
			if contains(instance.GetFinalizers(), backupFinalizer) {
				if err := r.finalizeBackup(logger, instance); err != nil {
					return ctrl.Result{}, err
				}
				instance.SetFinalizers(remove(instance.GetFinalizers(), backupFinalizer))
				if err := r.client.Update(ctx, instance); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		}
		if !contains(instance.GetFinalizers(), backupFinalizer) {
			if err := r.addFinalizer(logger, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	*/

	// Execute your backup logic (for example, creating or updating a Job).
	if err := r.create(logger, instance); err != nil {
		return ctrl.Result{}, err
	}

	// If no requeue is needed, return an empty result.
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// returns false if DistributedRedisCluster is ignored (not managed) by this operator.
			if !utils.ShoudManage(e.ObjectNew.GetAnnotations()) {
				return false
			}
			r.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).V(5).Info("Call UpdateFunc")
			// Ignore updates to CR status in which case metadata.Generation does not change
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				r.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).Info("Generation change return true",
					"old", e.ObjectOld, "new", e.ObjectNew)
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// returns false if DistributedRedisCluster is ignored (not managed) by this operator.
			if !utils.ShoudManage(e.Object.GetAnnotations()) {
				return false
			}
			r.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call DeleteFunc")
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// returns false if DistributedRedisCluster is ignored (not managed) by this operator.
			if !utils.ShoudManage(e.Object.GetAnnotations()) {
				return false
			}
			r.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).Info("Call CreateFunc")
			return true
		},
	}

	jobPred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			r.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).V(4).Info("Call Job UpdateFunc")
			if !utils.ShoudManage(e.ObjectNew.GetAnnotations()) {
				r.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).V(4).Info("Job UpdateFunc Not Manage")
				return false
			}
			newObj := e.ObjectNew.(*batch.Job)
			return isJobFinished(newObj)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if !utils.ShoudManage(e.Object.GetAnnotations()) {
				return false
			}
			job, ok := e.Object.(*batch.Job)
			if !ok {
				r.Log.Error(nil, "Invalid Job object")
				return false
			}
			if job.Status.Succeeded == 0 && job.Status.Failed <= utils.Int32(job.Spec.BackoffLimit) {
				return true
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			r.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).V(4).Info("Call Job CreateFunc")
			if !utils.ShoudManage(e.Object.GetAnnotations()) {
				r.Log.WithValues("namespace", e.Object.GetNamespace(), "name", e.Object.GetName()).V(4).Info("Job CreateFunc Not Manage")
				return false
			}
			job := e.Object.(*batch.Job)
			if job.Status.Succeeded > 0 || job.Status.Failed >= utils.Int32(job.Spec.BackoffLimit) {
				return true
			}
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.RedisClusterBackup{}, builder.WithPredicates(pred)).
		Owns(&batch.Job{}, builder.WithPredicates(jobPred)).
		Watches(
			&batch.Job{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(), mgr.GetRESTMapper(), &redisv1alpha1.RedisClusterBackup{}, handler.OnlyControllerOwner()),
			builder.WithPredicates(jobPred)).
		Complete(r)
}

// finalizeBackup contains cleanup logic for when a RedisClusterBackup is deleted.
func (r *RedisClusterBackupReconciler) finalizeBackup(b *redisv1alpha1.RedisClusterBackup) error {
	r.Log.Info("Successfully finalized RedisClusterBackup")
	return nil
}

// addFinalizer adds the backup finalizer to the resource.
func (r *RedisClusterBackupReconciler) addFinalizer(b *redisv1alpha1.RedisClusterBackup) error {
	r.Log.Info("Adding Finalizer for the backup")
	b.SetFinalizers(append(b.GetFinalizers(), backupFinalizer))
	if err := r.Client.Update(context.Background(), b); err != nil {
		r.Log.Error(err, "Failed to update RedisClusterBackup with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
