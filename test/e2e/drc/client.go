package drc

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
)

func NewClient(config *rest.Config) (client.Client, error) {
	// Create a new scheme rather than using the global scheme.
	sch := runtime.NewScheme()
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
