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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DistributedRedisClusterSpec defines the desired state of DistributedRedisCluster
type DistributedRedisClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Command          []string                      `json:"command,omitempty"`
	Env              []corev1.EnvVar               `json:"env,omitempty"`
	MasterSize       int32                         `json:"masterSize,omitempty"`
	ClusterReplicas  int32                         `json:"clusterReplicas,omitempty"`
	ServiceName      string                        `json:"serviceName,omitempty"`
	Config           map[string]string             `json:"config,omitempty"`
	// Set RequiredAntiAffinity to force the master-slave node anti-affinity.
	RequiredAntiAffinity     bool                         `json:"requiredAntiAffinity,omitempty"`
	Affinity                 *corev1.Affinity             `json:"affinity,omitempty"`
	NodeSelector             map[string]string            `json:"nodeSelector,omitempty"`
	ToleRations              []corev1.Toleration          `json:"toleRations,omitempty"`
	SecurityContext          *corev1.PodSecurityContext   `json:"securityContext,omitempty"`
	ContainerSecurityContext *corev1.SecurityContext      `json:"containerSecurityContext,omitempty"`
	Annotations              map[string]string            `json:"annotations,omitempty"`
	Storage                  *RedisStorage                `json:"storage,omitempty"`
	Resources                *corev1.ResourceRequirements `json:"resources,omitempty"`
	PasswordSecret           *corev1.LocalObjectReference `json:"passwordSecret,omitempty"`
	Monitor                  *AgentSpec                   `json:"monitor,omitempty"`
	Init                     *InitSpec                    `json:"init,omitempty"`
}

type AgentSpec struct {
	Image      string          `json:"image,omitempty"`
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
	// Arguments to the entrypoint.
	// The docker image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax
	// can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,
	// regardless of whether the variable exists or not.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Args []string `json:"args,omitempty"`
	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// Compute Resources required by exporter container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
}

type PrometheusSpec struct {
	// Port number for the exporter sidecar.
	Port int32 `json:"port,omitempty"`

	// Namespace of Prometheus. Service monitors will be created in this namespace.
	Namespace string `json:"namespace,omitempty"`
	// Labels are key value pairs that is used to select Prometheus instance via ServiceMonitor labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Interval at which metrics should be scraped
	Interval string `json:"interval,omitempty"`
	//Annotations map[string]string `json:"annotations,omitempty"`
}

type InitSpec struct {
	BackupSource *BackupSourceSpec `json:"backupSource,omitempty"`
}

type BackupSourceSpec struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// Arguments to the restore job
	Args []string `json:"args,omitempty"`
}

// RedisStorage defines the structure used to store the Redis Data
type RedisStorage struct {
	Size        resource.Quantity `json:"size"`
	Type        StorageType       `json:"type"`
	Class       string            `json:"class"`
	DeleteClaim bool              `json:"deleteClaim,omitempty"`
}

// DistributedRedisClusterStatus defines the observed state of DistributedRedisCluster
type DistributedRedisClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status               ClusterStatus      `json:"status"`
	Reason               string             `json:"reason,omitempty"`
	NumberOfMaster       int32              `json:"numberOfMaster,omitempty"`
	MinReplicationFactor int32              `json:"minReplicationFactor,omitempty"`
	MaxReplicationFactor int32              `json:"maxReplicationFactor,omitempty"`
	NodesPlacement       NodesPlacementInfo `json:"nodesPlacementInfo,omitempty"`
	Nodes                []RedisClusterNode `json:"nodes"`

	// +optional
	Restore Restore `json:"restore"`
}

type Restore struct {
	Phase  RestorePhase        `json:"phase,omitempty"`
	Backup *RedisClusterBackup `json:"backup,omitempty"`
}

// RedisClusterNode represent a RedisCluster Node
type RedisClusterNode struct {
	ID          string    `json:"id"`
	Role        RedisRole `json:"role"`
	IP          string    `json:"ip"`
	Port        string    `json:"port"`
	Slots       []string  `json:"slots,omitempty"`
	MasterRef   string    `json:"masterRef,omitempty"`
	PodName     string    `json:"podName"`
	NodeName    string    `json:"nodeName"`
	StatefulSet string    `json:"statefulSet"`
}

//+kubebuilder:resource:shortName="drc"
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods;secrets;endpoints;persistentvolumeclaims,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=configmaps;pods/exec;secrets;services;events;persistentvolumeclaims,verbs=create;get;list;patch;update;watch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;delete;patch

// DistributedRedisCluster is the Schema for the DistributedRedisClusters API
type DistributedRedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DistributedRedisClusterSpec   `json:"spec,omitempty"`
	Status DistributedRedisClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DistributedRedisClusterList contains a list of DistributedRedisCluster
type DistributedRedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DistributedRedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DistributedRedisCluster{}, &DistributedRedisClusterList{})
}
