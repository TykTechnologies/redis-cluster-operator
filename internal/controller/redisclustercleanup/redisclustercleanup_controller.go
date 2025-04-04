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

package redisclustercleanup

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
	"github.com/TykTechnologies/redis-cluster-operator/internal/k8sutil"
)

// RedisClusterCleanupReconciler reconciles a RedisClusterCleanup object
type RedisClusterCleanupReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	crController k8sutil.ICustomResource

	Log logr.Logger
}

// NewReconcileRedisClusterCleanup creates a new instance of the reconciler.
func NewReconcileRedisClusterCleanup(mgr ctrl.Manager) *RedisClusterCleanupReconciler {
	// If your direct client is needed, you can create it here (or use mgr.GetClient()).
	return &RedisClusterCleanupReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		Log:          ctrl.Log.WithName("controllers").WithName("RedisClusterCleanup"),
		crController: k8sutil.NewCRControl(mgr.GetClient()),
	}
}

// +kubebuilder:rbac:groups=redis.kun,resources=redisclustercleanups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redis.kun,resources=redisclustercleanups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redis.kun,resources=redisclustercleanups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisClusterCleanup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *RedisClusterCleanupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)

	redisClusterCleanup := &redisv1alpha1.RedisClusterCleanup{}
	if err := r.Get(ctx, req.NamespacedName, redisClusterCleanup); err != nil {
		logger.Info("Failed to get RedisClusterCleanup", "error", err.Error())
		return ctrl.Result{RequeueAfter: 10 * time.Second}, client.IgnoreNotFound(err)
	}

	if redisClusterCleanup.Spec.Suspend {
		logger.Info("RedisClusterCleanup suspend, skipping ...")
		return ctrl.Result{}, nil
	}

	distributedRedisClusters := redisv1alpha1.DistributedRedisClusterList{}

	// Create the ListOptions with the filter for namespaces
	listOptions := &client.ListOptions{
		Namespace: strings.Join(redisClusterCleanup.Spec.Namespaces, ","), // You can join the namespaces with commas
	}

	if err := r.List(ctx, &distributedRedisClusters, listOptions); err != nil {
		logger.Info("Failed to list DistributedRedisClusters", "error", err.Error())
		return ctrl.Result{RequeueAfter: 10 * time.Second}, client.IgnoreNotFound(err)
	}

	redisClusterCleanup.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}

	for _, distributedRedisCluster := range distributedRedisClusters.Items {
		logger.V(3).Info("Checking for expired keys", "cluster", distributedRedisCluster.Name)
		if distributedRedisCluster.Status.Reason != "OK" {
			logger.Info("Skip the cluster since its status is not OK ", "cluster",
				distributedRedisCluster.Name, "status", distributedRedisCluster.Status.Status,
				"Namespace", distributedRedisCluster.Namespace)
			continue
		}
		masterNum := int(distributedRedisCluster.Spec.MasterSize)
		redisHosts := []string{}
		for _, node := range distributedRedisCluster.Status.Nodes {
			if node.Role == redisv1alpha1.RedisClusterNodeRoleMaster {
				redisHosts = append(redisHosts, node.IP)
				// Optional: Stop when you've collected the expected number of masters.
				if len(redisHosts) == masterNum {
					break
				}
			}
		}

		// Read the secret containing the Redis password.
		// Ensure you are referencing the correct namespace where the secret exists.
		secret := &corev1.Secret{}
		secretName := types.NamespacedName{
			Namespace: distributedRedisCluster.Namespace, // adjust if the secret is in a different namespace
			Name:      distributedRedisCluster.Spec.PasswordSecret.Name,
		}
		if err := r.Get(ctx, secretName, secret); err != nil {
			logger.Error(err, "Failed to get secret", "secret", secretName)
			continue // or handle the error as appropriate for your use case
		}

		passwordBytes, exists := secret.Data["password"]
		if !exists {
			logger.Error(nil, "Password key not found in secret", "secret", secretName)
			continue // or handle the missing key error appropriately
		}

		redisPassword := string(passwordBytes)
		logger.V(3).Info("Successfully retrieved Redis password", "cluster", distributedRedisCluster.Name)

		logger.Info("Cleaning", "DRC namespace", distributedRedisCluster.Namespace, "DRC name", distributedRedisCluster.Name)

		var wg sync.WaitGroup
		// Create a goroutine for each host.
		for _, host := range redisHosts {
			host = strings.TrimSpace(host)
			if host == "" {
				continue
			}
			wg.Add(1)
			go func(host string) {
				defer wg.Done()
				processHost(host, "6379", redisPassword, int64(redisClusterCleanup.Spec.ExpiredThreshold), logger)
			}(host)
		}
		wg.Wait()
	}

	redisClusterCleanup.Status.LastSuccessfulTime = &metav1.Time{Time: time.Now()}
	if err := r.crController.UpdateCRStatus(redisClusterCleanup); err != nil {
		logger.Error(err, "Failed to update the status", "redisClusterCleanup", redisClusterCleanup)
	}

	// Parse the cron schedule from your custom resource field
	scheduleStr := redisClusterCleanup.Spec.Schedule
	// Create a parser that understands the standard 5-field cron syntax.
	// If your schedule uses seconds, adjust the parser flags accordingly.
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(scheduleStr)
	if err != nil {
		logger.Error(err, "Failed to parse cron schedule", "schedule", scheduleStr)
		// Return a default requeue interval if schedule parsing fails
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Calculate the duration until the next scheduled time
	now := time.Now()
	nextRun := sched.Next(now)
	durationUntilNext := nextRun.Sub(now)
	logger.Info("Next reconcile scheduled", "duration", durationUntilNext)

	return ctrl.Result{RequeueAfter: durationUntilNext}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterCleanupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.RedisClusterCleanup{}).
		Complete(r)
}
