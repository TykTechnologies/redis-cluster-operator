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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisClusterCleanupSpec defines the desired state of RedisClusterCleanup
type RedisClusterCleanupSpec struct {
	// Schedule is a cron expression to run the cleanup job.
	// +kubebuilder:validation:Pattern=`^(\S+\s+){4}\S+$`
	Schedule string `json:"schedule"`

	// +kubebuilder:default:=false
	Suspend bool `json:"suspend,omitempty"`

	// ExpiredThreshold defines the minimum number of expired keys that triggers a cleanup.
	// +kubebuilder:default:=200
	ExpiredThreshold int `json:"expiredThreshold,omitempty"`

	// ExpiredThreshold defines the minimum number of expired keys that triggers a cleanup.
	// +kubebuilder:default:=200
	ScanBatchSize int `json:"scanBatchSize,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Namespaces []string `json:"namespaces"`

	// KeyPatterns holds one or more patterns for SCAN operations.
	// For example, ["apikey-*", "session-*"]
	KeyPatterns []string `json:"keyPatterns,omitempty"`

	// ExpirationRegexes holds one or more regexes to extract the expiration value.
	// For example: ["\"expires\":\\s*(\\d+)"]
	ExpirationRegexes []string `json:"expirationRegexes,omitempty"`

	// SkipPatterns holds substrings or patterns that if found in a key's value will skip deletion.
	// For example, ["TykJWTSessionID"]
	SkipPatterns []string `json:"skipPatterns,omitempty"`
}

// RedisClusterCleanupStatus defines the observed state of RedisClusterCleanup
type RedisClusterCleanupStatus struct {
	// LastScheduleTime is the last time the CronJob was scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
	// LastSuccessfulTime is the last time the CronJob completed successfully.
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`
}

// +kubebuilder:resource:shortName="drcc"
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule",description="Cleanup schedule"
// +kubebuilder:printcolumn:name="Suspend",type="boolean",JSONPath=".spec.suspend",description="Whether the Cleaner is currently suspended (True/False)."
// +kubebuilder:printcolumn:name="LastSuccessfulTime",type="date",JSONPath=".status.lastSuccessfulTime",description="The last time the Cleaner completed successfully"

// RedisClusterCleanup is the Schema for the redisclustercleanups API
type RedisClusterCleanup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisClusterCleanupSpec   `json:"spec,omitempty"`
	Status RedisClusterCleanupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisClusterCleanupList contains a list of RedisClusterCleanup
type RedisClusterCleanupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisClusterCleanup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisClusterCleanup{}, &RedisClusterCleanupList{})
}
