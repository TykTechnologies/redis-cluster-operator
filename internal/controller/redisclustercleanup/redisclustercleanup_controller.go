/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
...
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
	ctrlController "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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

func (r *RedisClusterCleanupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)

	// Fetch the RedisClusterCleanup custom resource.
	redisClusterCleanup := &redisv1alpha1.RedisClusterCleanup{}
	if err := r.Get(ctx, req.NamespacedName, redisClusterCleanup); err != nil {
		logger.Info("Failed to get RedisClusterCleanup", "error", err.Error())
		return ctrl.Result{RequeueAfter: 10 * time.Second}, client.IgnoreNotFound(err)
	}

	if redisClusterCleanup.Spec.Suspend {
		logger.Info("RedisClusterCleanup suspended, skipping ...")
		return ctrl.Result{}, nil
	}

	// List DistributedRedisClusters. Note: Adjust the listing logic if you require filtering across multiple namespaces.
	distributedRedisClusters := redisv1alpha1.DistributedRedisClusterList{}
	listOptions := &client.ListOptions{
		Namespace: strings.Join(redisClusterCleanup.Spec.Namespaces, ","),
	}
	if err := r.List(ctx, &distributedRedisClusters, listOptions); err != nil {
		logger.Info("Failed to list DistributedRedisClusters", "error", err.Error())
		return ctrl.Result{RequeueAfter: 10 * time.Second}, client.IgnoreNotFound(err)
	}

	// Update the last schedule time.
	redisClusterCleanup.Status.LastScheduleTime = &metav1.Time{Time: time.Now()}
	if err := r.crController.UpdateCRStatus(redisClusterCleanup); err != nil {
		logger.Error(err, "Failed to update status", "redisClusterCleanup", redisClusterCleanup)
	}

	// Semaphore to limit concurrent cluster processing to 5 at a time.
	semaphore := make(chan struct{}, 5)
	var wgClusters sync.WaitGroup

	// Process each cluster.
	for _, distributedRedisCluster := range distributedRedisClusters.Items {
		// Acquire a semaphore token.
		semaphore <- struct{}{}
		wgClusters.Add(1)

		// Process each cluster concurrently.
		go func(cluster redisv1alpha1.DistributedRedisCluster) {
			defer wgClusters.Done()
			defer func() { <-semaphore }() // Release the semaphore slot.

			// Only process healthy clusters.
			if cluster.Status.Status != redisv1alpha1.ClusterStatusOK {
				logger.Info("Skipping unhealthy cluster", "cluster", cluster.Name, "status", cluster.Status.Status)
				return
			}

			// Collect master node IPs up to the specified master count.
			masterNum := int(cluster.Spec.MasterSize)
			var redisHosts []string
			for _, node := range cluster.Status.Nodes {
				if node.Role == redisv1alpha1.RedisClusterNodeRoleMaster {
					redisHosts = append(redisHosts, node.IP)
					if len(redisHosts) == masterNum {
						break
					}
				}
			}

			// Retrieve the secret containing the Redis password.
			secret := &corev1.Secret{}
			secretName := types.NamespacedName{
				Namespace: cluster.Namespace, // adjust if secret is in a different namespace
				Name:      cluster.Spec.PasswordSecret.Name,
			}
			if err := r.Get(ctx, secretName, secret); err != nil {
				logger.Error(err, "Failed to get secret", "secret", secretName)
				return
			}

			passwordBytes, exists := secret.Data["password"]
			if !exists {
				logger.Error(nil, "Password key not found in secret", "secret", secretName)
				return
			}
			redisPassword := string(passwordBytes)
			logger.V(3).Info("Successfully retrieved Redis password", "cluster", cluster.Name)

			logger.Info("Cleaning cluster", "Namespace", cluster.Namespace, "Name", cluster.Name)

			// Process each host concurrently within this cluster.
			var wgHosts sync.WaitGroup
			for _, host := range redisHosts {
				host = strings.TrimSpace(host)
				if host == "" {
					continue
				}
				wgHosts.Add(1)
				go func(host string) {
					defer wgHosts.Done()
					// processHost is your function that handles the individual host cleanup.
					processHost(host, "6379", redisPassword, redisClusterCleanup.Spec, logger)
				}(host)
			}
			wgHosts.Wait()
		}(distributedRedisCluster)
	}
	wgClusters.Wait()

	// Update the status after successful processing.
	redisClusterCleanup.Status.LastSuccessfulTime = &metav1.Time{Time: time.Now()}
	redisClusterCleanup.Status.Succeed += 1
	if err := r.crController.UpdateCRStatus(redisClusterCleanup); err != nil {
		logger.Error(err, "Failed to update status", "redisClusterCleanup", redisClusterCleanup)
	}

	// Parse the cron schedule to determine when to requeue.
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(redisClusterCleanup.Spec.Schedule)
	if err != nil {
		logger.Error(err, "Failed to parse cron schedule", "schedule", redisClusterCleanup.Spec.Schedule)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	now := time.Now()
	nextRun := sched.Next(now)
	durationUntilNext := nextRun.Sub(now)
	logger.Info("Next reconcile scheduled", "duration", durationUntilNext)

	return ctrl.Result{RequeueAfter: durationUntilNext}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterCleanupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			r.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).V(5).Info("Call UpdateFunc")
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				r.Log.WithValues("namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName()).Info("Generation changed, processing update")
				return true
			}
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.RedisClusterCleanup{}).
		WithEventFilter(pred).
		WithOptions(ctrlController.Options{}).
		Complete(r)
}
