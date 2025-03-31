package k8sutil

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ICronJobControl define the interface that uses to create, update, and delete CronJobs.
type ICronJobControl interface {
	CreateCronJob(*batchv1.CronJob) error
	UpdateCronJob(*batchv1.CronJob) error
	DeleteCronJob(*batchv1.CronJob) error
	GetCronJob(namespace, name string) (*batchv1.CronJob, error)
	ListCronJobByLabels(namespace string, labs client.MatchingLabels) (*batchv1.CronJobList, error)
}

type CronJobController struct {
	client client.Client
}

// NewCronJobController creates a concrete implementation of the
// IJobControl.
func NewCronJobController(client client.Client) ICronJobControl {
	return &CronJobController{client: client}
}

func (c *CronJobController) CreateCronJob(cronjob *batchv1.CronJob) error {
	return c.client.Create(context.TODO(), cronjob)
}

func (c CronJobController) UpdateCronJob(cronjob *batchv1.CronJob) error {
	return c.client.Update(context.TODO(), cronjob)
}

func (c CronJobController) DeleteCronJob(cronjob *batchv1.CronJob) error {
	return c.client.Delete(context.TODO(), cronjob)
}

func (c CronJobController) GetCronJob(namespace, name string) (*batchv1.CronJob, error) {
	cronjob := &batchv1.CronJob{}
	err := c.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, cronjob)
	return cronjob, err
}

func (c CronJobController) ListCronJobByLabels(namespace string, labels client.MatchingLabels) (*batchv1.CronJobList, error) {
	cronjobList := &batchv1.CronJobList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
		labels,
	}
	err := c.client.List(context.TODO(), cronjobList, opts...)
	if err != nil {
		return nil, err
	}

	return cronjobList, nil
}
