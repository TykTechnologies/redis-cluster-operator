package drc

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
)

func NewClient(config *rest.Config) (client.Client, error) {
	// Create a new scheme rather than using the global scheme.
	sch := runtime.NewScheme()
	_ = corev1.AddToScheme(sch)
	_ = rbacv1.AddToScheme(sch)
	_ = appsv1.AddToScheme(sch)

	// Add your API types to the scheme and check for errors.
	if err := redisv1alpha1.AddToScheme(sch); err != nil {
		return nil, err
	}

	// Set up client options with your custom scheme and mapper.
	options := client.Options{
		Scheme: sch,
	}

	// Create and return a new client instance.
	cli, err := client.New(config, options)
	if err != nil {
		return nil, err
	}
	return cli, nil
}
