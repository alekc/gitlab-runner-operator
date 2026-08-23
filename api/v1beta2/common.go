/*
Copyright 2020 Alexander Chernov

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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// ConditionReady is the condition type reported on the runner status.
const ConditionReady = "Ready"

// Defaults for ConcurrencyLimits. The CRD markers below carry the same values
// as literals, and the generator floors on these constants: helm does not
// update CRDs on upgrade, so a new binary can meet an object the API server
// never defaulted.
const (
	DefaultRunnerLimit        = 10
	DefaultRequestConcurrency = 3
)

// ConcurrencyLimits are the per-entry budgets, embedded inline in both kinds so
// the two cannot drift apart.
type ConcurrencyLimits struct {
	// Limit caps the jobs this entry executes at once. The effective ceiling is
	// the lower of Limit and Concurrent, so raising Concurrent alone does
	// nothing past this value.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	Limit int `json:"limit,omitempty"`

	// RequestConcurrency caps job requests in flight to GitLab, not jobs
	// running. At 1 an entry acquires work one round trip at a time, so a
	// raised Limit fills slowly.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	RequestConcurrency int `json:"request_concurrency,omitempty"`
}

// EffectiveLimit returns the job cap, falling back to the default when the API
// server did not apply one. Mirrors EffectiveNamespace: one rule, both paths.
func (c ConcurrencyLimits) EffectiveLimit() int {
	if c.Limit <= 0 {
		return DefaultRunnerLimit
	}
	return c.Limit
}

// EffectiveRequestConcurrency returns the polling cap, with the same fallback.
func (c ConcurrencyLimits) EffectiveRequestConcurrency() int {
	if c.RequestConcurrency <= 0 {
		return DefaultRequestConcurrency
	}
	return c.RequestConcurrency
}

// defaultRunnerResources are the resource requests/limits applied to the runner
// manager container when the spec does not override them.
func defaultRunnerResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
}

// defaultRunnerSecurityContext hardens the runner manager container without
// forcing runAsNonRoot (the gitlab-runner image's user is image-dependent).
func defaultRunnerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
}

// setReadyCondition records the Ready condition idempotently (the helper only
// bumps LastTransitionTime when the status actually flips).
func setReadyCondition(conditions *[]metav1.Condition, generation int64, ready bool, reason, message string) {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
	})
}
