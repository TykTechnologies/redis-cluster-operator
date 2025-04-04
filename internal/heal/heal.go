package heal

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/TykTechnologies/redis-cluster-operator/internal/k8sutil"
)

type CheckAndHeal struct {
	Logger     logr.Logger
	PodControl k8sutil.IPodControl
	Pods       []*corev1.Pod
	DryRun     bool
}
