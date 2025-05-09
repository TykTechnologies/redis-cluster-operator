package k8sutil

import (
	"context"

	policyv11 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IPodDisruptionBudgetControl defines the interface that uses to create, update, and delete PodDisruptionBudgets.
type IPodDisruptionBudgetControl interface {
	// CreatePodDisruptionBudget creates a PodDisruptionBudget in a DistributedRedisCluster.
	CreatePodDisruptionBudget(*policyv11.PodDisruptionBudget) error
	// UpdatePodDisruptionBudget updates a PodDisruptionBudget in a DistributedRedisCluster.
	UpdatePodDisruptionBudget(*policyv11.PodDisruptionBudget) error
	// DeletePodDisruptionBudget deletes a PodDisruptionBudget in a DistributedRedisCluster.
	DeletePodDisruptionBudget(*policyv11.PodDisruptionBudget) error
	DeletePodDisruptionBudgetByName(namespace, name string) error
	// GetPodDisruptionBudget get PodDisruptionBudget in a DistributedRedisCluster.
	GetPodDisruptionBudget(namespace, name string) (*policyv11.PodDisruptionBudget, error)
}

type PodDisruptionBudgetController struct {
	client client.Client
}

// NewPodDisruptionBudgetController creates a concrete implementation of the
// IPodDisruptionBudgetControl.
func NewPodDisruptionBudgetController(client client.Client) IPodDisruptionBudgetControl {
	return &PodDisruptionBudgetController{client: client}
}

// CreatePodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) CreatePodDisruptionBudget(pb *policyv11.PodDisruptionBudget) error {
	return s.client.Create(context.TODO(), pb)
}

// UpdatePodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) UpdatePodDisruptionBudget(pb *policyv11.PodDisruptionBudget) error {
	return s.client.Update(context.TODO(), pb)
}

// DeletePodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) DeletePodDisruptionBudget(pb *policyv11.PodDisruptionBudget) error {
	return s.client.Delete(context.TODO(), pb)
}

func (s *PodDisruptionBudgetController) DeletePodDisruptionBudgetByName(namespace, name string) error {
	pdb, err := s.GetPodDisruptionBudget(namespace, name)
	if err != nil {
		return err
	}
	return s.DeletePodDisruptionBudget(pdb)
}

// GetPodDisruptionBudget implement the IPodDisruptionBudgetControl.Interface.
func (s *PodDisruptionBudgetController) GetPodDisruptionBudget(namespace, name string) (*policyv11.PodDisruptionBudget, error) {
	pb := &policyv11.PodDisruptionBudget{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pb)
	return pb, err
}
