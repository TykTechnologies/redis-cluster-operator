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

package distributedrediscluster

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	ctrlController "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
	"github.com/TykTechnologies/redis-cluster-operator/internal/config"
	"github.com/TykTechnologies/redis-cluster-operator/internal/controller"
	"github.com/TykTechnologies/redis-cluster-operator/internal/exec"
	"github.com/TykTechnologies/redis-cluster-operator/internal/heal"
	"github.com/TykTechnologies/redis-cluster-operator/internal/k8sutil"
	clustermanger "github.com/TykTechnologies/redis-cluster-operator/internal/manager"
	"github.com/TykTechnologies/redis-cluster-operator/internal/redisutil"
	"github.com/TykTechnologies/redis-cluster-operator/internal/resources/statefulsets"
	"github.com/TykTechnologies/redis-cluster-operator/internal/utils"
)

// DistributedRedisClusterReconciler reconciles a DistributedRedisCluster object
type DistributedRedisClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ensurer clustermanger.IEnsureResource
	checker clustermanger.ICheck

	// Remote execution dependency.
	execer exec.IExec

	// Helper controllers.
	statefulSetController k8sutil.IStatefulSetControl
	serviceController     k8sutil.IServiceControl
	pdbController         k8sutil.IPodDisruptionBudgetControl
	pvcController         k8sutil.IPvcControl
	crController          k8sutil.ICustomResource

	Log logr.Logger
}

// NewDistributedRedisClusterReconciler returns a new reconciler instance configured with its dependencies.
func NewDistributedRedisClusterReconciler(mgr ctrl.Manager) (*DistributedRedisClusterReconciler, error) {
	// Create a REST client for a Pod GVK for remote execution.
	gvk := runtimeschema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}

	config := mgr.GetConfig()
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}

	restClient, err := apiutil.RESTClientForGVK(gvk, false, mgr.GetConfig(), serializer.NewCodecFactory(scheme.Scheme), httpClient)
	if err != nil {
		return nil, err
	}
	execer := exec.NewRemoteExec(restClient, mgr.GetConfig(), ctrl.Log.WithName("remoteexec"))

	return &DistributedRedisClusterReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		execer:                execer,
		statefulSetController: k8sutil.NewStatefulSetController(mgr.GetClient()),
		serviceController:     k8sutil.NewServiceController(mgr.GetClient()),
		pdbController:         k8sutil.NewPodDisruptionBudgetController(mgr.GetClient()),
		pvcController:         k8sutil.NewPvcController(mgr.GetClient()),
		crController:          k8sutil.NewCRControl(mgr.GetClient()),
		ensurer:               clustermanger.NewEnsureResource(mgr.GetClient(), ctrl.Log.WithName("ensurer")),
		checker:               clustermanger.NewCheck(mgr.GetClient()),
		Log:                   ctrl.Log.WithName("controller").WithName("distributedrediscluster"),
	}, nil
}

// +kubebuilder:rbac:groups=redis.kun,resources=distributedredisclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redis.kun,resources=distributedredisclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redis.kun,resources=distributedredisclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DistributedRedisCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *DistributedRedisClusterReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {

	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling DistributedRedisCluster")

	// Fetch the DistributedRedisCluster instance
	instance := &redisv1alpha1.DistributedRedisCluster{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ctx := &syncContext{
		cluster:   instance,
		reqLogger: reqLogger,
	}

	err = r.ensureCluster(ctx)
	if err != nil {
		switch controller.GetType(err) {
		case controller.StopRetry:
			reqLogger.Info("invalid", "err", err)
			return reconcile.Result{}, nil
		}
		reqLogger.WithValues("err", err).Info("ensureCluster")
		newStatus := instance.Status.DeepCopy()
		SetClusterScaling(newStatus, err.Error())
		r.updateClusterIfNeed(instance, newStatus, reqLogger)
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	matchLabels := getLabels(instance)
	redisClusterPods, err := r.statefulSetController.GetStatefulSetPodsByLabels(instance.Namespace, matchLabels)
	if err != nil {
		return reconcile.Result{}, controller.Kubernetes.Wrap(err, "GetStatefulSetPods")
	}

	ctx.pods = clusterPods(redisClusterPods.Items)
	reqLogger.V(6).Info("debug cluster pods", "", ctx.pods)
	ctx.healer = clustermanger.NewHealer(&heal.CheckAndHeal{
		Logger:     reqLogger,
		PodControl: k8sutil.NewPodController(r.Client),
		Pods:       ctx.pods,
		DryRun:     false,
	})
	err = r.waitPodReady(ctx)
	if err != nil {
		switch controller.GetType(err) {
		case controller.Kubernetes:
			return reconcile.Result{}, err
		}
		reqLogger.WithValues("err", err).Info("waitPodReady")
		newStatus := instance.Status.DeepCopy()
		SetClusterScaling(newStatus, err.Error())
		r.updateClusterIfNeed(instance, newStatus, reqLogger)
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	password, err := statefulsets.GetClusterPassword(r.Client, instance)
	if err != nil {
		return reconcile.Result{}, controller.Kubernetes.Wrap(err, "getClusterPassword")
	}

	admin, err := newRedisAdmin(ctx.pods, password, config.RedisConf(), reqLogger)
	if err != nil {
		return reconcile.Result{}, controller.Redis.Wrap(err, "newRedisAdmin")
	}
	defer admin.Close()

	clusterInfos, err := admin.GetClusterInfos()
	if err != nil {
		if clusterInfos.Status == redisutil.ClusterInfosPartial {
			return reconcile.Result{}, controller.Redis.Wrap(err, "GetClusterInfos")
		}
	}

	requeue, err := ctx.healer.Heal(instance, clusterInfos, admin)
	if err != nil {
		return reconcile.Result{}, controller.Redis.Wrap(err, "Heal")
	}
	if requeue {
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	ctx.admin = admin
	ctx.clusterInfos = clusterInfos
	err = r.waitForClusterJoin(ctx)
	if err != nil {
		switch controller.GetType(err) {
		case controller.Requeue:
			reqLogger.WithValues("err", err).Info("requeue")
			return reconcile.Result{RequeueAfter: requeueAfter}, nil
		}
		newStatus := instance.Status.DeepCopy()
		SetClusterFailed(newStatus, err.Error())
		r.updateClusterIfNeed(instance, newStatus, reqLogger)
		return reconcile.Result{}, err
	}

	// mark .Status.Restore.Phase = RestorePhaseRestart, will
	// remove init container and restore volume that referenced in stateulset for
	// dump RDB file from backup, then the redis master node will be restart.
	if instance.IsRestoreFromBackup() && instance.IsRestoreRunning() {
		reqLogger.Info("update restore redis cluster cr")
		instance.Status.Restore.Phase = redisv1alpha1.RestorePhaseRestart
		if err := r.crController.UpdateCRStatus(instance); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.ensurer.UpdateRedisStatefulsets(instance, getLabels(instance)); err != nil {
			return reconcile.Result{}, err
		}
		waiter := &waitStatefulSetUpdating{
			name:                  "waitMasterNodeRestarting",
			timeout:               60 * time.Second,
			tick:                  5 * time.Second,
			statefulSetController: r.statefulSetController,
			cluster:               instance,
		}
		if err := waiting(waiter, ctx.reqLogger); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// restore succeeded, then update cr and wait for the next Reconcile loop
	if instance.IsRestoreFromBackup() && instance.IsRestoreRestarting() {
		reqLogger.Info("update restore redis cluster cr")
		instance.Status.Restore.Phase = redisv1alpha1.RestorePhaseSucceeded
		if err := r.crController.UpdateCRStatus(instance); err != nil {
			return reconcile.Result{}, err
		}
		// set ClusterReplicas = Backup.Status.ClusterReplicas,
		// next Reconcile loop the statefulSet's replicas will increase by ClusterReplicas, then start the slave node
		instance.Spec.ClusterReplicas = instance.Status.Restore.Backup.Status.ClusterReplicas
		if err := r.crController.UpdateCR(instance); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := admin.SetConfigIfNeed(instance.Spec.Config); err != nil {
		return reconcile.Result{}, controller.Redis.Wrap(err, "SetConfigIfNeed")
	}

	status := buildClusterStatus(clusterInfos, ctx.pods, instance, reqLogger)
	if is := r.isScalingDown(instance, reqLogger); is {
		SetClusterRebalancing(status, "scaling down")
	}
	reqLogger.V(4).Info("buildClusterStatus", "status", status)
	err = r.updateClusterIfNeed(instance, status, reqLogger)
	if err != nil {
		reqLogger.Error(err, "problem with update status")
	}

	instance.Status = *status
	if needClusterOperation(instance, reqLogger) {
		reqLogger.Info(">>>>>> clustering")
		err = r.syncCluster(ctx)
		if err != nil {
			newStatus := instance.Status.DeepCopy()
			SetClusterFailed(newStatus, err.Error())
			err = r.updateClusterIfNeed(instance, newStatus, reqLogger)
			if err != nil {
				reqLogger.Error(err, "problem with update cluster")
			}
			return reconcile.Result{}, err
		}
	}

	newClusterInfos, err := admin.GetClusterInfos()
	if err != nil {
		if clusterInfos.Status == redisutil.ClusterInfosPartial {
			return reconcile.Result{}, controller.Redis.Wrap(err, "GetClusterInfos")
		}
	}
	newStatus := buildClusterStatus(newClusterInfos, ctx.pods, instance, reqLogger)
	SetClusterOK(newStatus, "OK")
	r.updateClusterIfNeed(instance, newStatus, reqLogger)

	return ctrl.Result{}, nil
}

func (r *DistributedRedisClusterReconciler) isScalingDown(cluster *redisv1alpha1.DistributedRedisCluster, reqLogger logr.Logger) bool {
	stsList, err := r.statefulSetController.ListStatefulSetByLabels(cluster.Namespace, getLabels(cluster))
	if err != nil {
		reqLogger.Error(err, "ListStatefulSetByLabels")
		return false
	}
	if len(stsList.Items) > int(cluster.Spec.MasterSize) {
		return true
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *DistributedRedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// If you still need your execer, you can create it outside or embed its initialization in your struct.
	// Setup your predicates as before.
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.DistributedRedisCluster{}).
		WithEventFilter(pred).
		WithOptions(ctrlController.Options{}).
		Complete(r)
}
