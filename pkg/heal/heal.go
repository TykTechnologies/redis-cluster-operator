package heal

import (
	"github.com/TykTechnologies/redis-cluster-operator/pkg/k8sutil"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

type CheckAndHeal struct {
	Logger     logr.Logger
	PodControl k8sutil.IPodControl
	Pods       []*corev1.Pod
	DryRun     bool
}
