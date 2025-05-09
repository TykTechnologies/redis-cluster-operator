package services

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
)

// NewHeadLessSvcForCR creates a new headless service for the given Cluster.
func NewHeadLessSvcForCR(cluster *redisv1alpha1.DistributedRedisCluster, name string, labels map[string]string) *corev1.Service {
	clientPort := corev1.ServicePort{Name: "client", Port: 6379}
	gossipPort := corev1.ServicePort{Name: "gossip", Port: 16379}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            name,
			Namespace:       cluster.Namespace,
			OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{clientPort, gossipPort},
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func NewSvcForCR(cluster *redisv1alpha1.DistributedRedisCluster, name string, labels map[string]string) *corev1.Service {
	var ports []corev1.ServicePort
	clientPort := corev1.ServicePort{Name: "client", Port: 6379}
	gossipPort := corev1.ServicePort{Name: "gossip", Port: 16379}
	if cluster.Spec.Monitor == nil {
		ports = append(ports, clientPort, gossipPort)
	} else {
		ports = append(ports, clientPort, gossipPort,
			corev1.ServicePort{Name: "prom-http", Port: cluster.Spec.Monitor.Prometheus.Port})
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            name,
			Namespace:       cluster.Namespace,
			OwnerReferences: redisv1alpha1.DefaultOwnerReferences(cluster),
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: labels,
		},
	}

	return svc
}
