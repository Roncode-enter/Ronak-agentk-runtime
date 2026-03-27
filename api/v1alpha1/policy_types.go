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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyType defines the enforcement engine used by the policy.
// +kubebuilder:validation:Enum=ebpf;opa;hybrid
type PolicyType string

const (
	PolicyTypeEBPF   PolicyType = "ebpf"
	PolicyTypeOPA    PolicyType = "opa"
	PolicyTypeHybrid PolicyType = "hybrid"
)

// PolicyAction defines what happens when a policy rule is triggered.
// +kubebuilder:validation:Enum=block;audit;alert
type PolicyAction string

const (
	PolicyActionBlock PolicyAction = "block"
	PolicyActionAudit PolicyAction = "audit"
	PolicyActionAlert PolicyAction = "alert"
)

// EnforcementMode defines whether the policy actively blocks or only observes.
// +kubebuilder:validation:Enum=enforcing;permissive
type EnforcementMode string

const (
	EnforcementModeEnforcing  EnforcementMode = "enforcing"
	EnforcementModePermissive EnforcementMode = "permissive"
)

// EBPFRuleConfig holds configuration for an eBPF-based policy rule.
type EBPFRuleConfig struct {
	// Program identifies the type of eBPF program to attach.
	// +kubebuilder:validation:Enum=network-egress;syscall-filter;file-access
	Program string `json:"program"`

	// AllowedCIDRs lists network ranges the agent is allowed to reach.
	// Only applicable for network-egress programs.
	// +optional
	AllowedCIDRs []string `json:"allowedCIDRs,omitempty"`

	// DeniedPorts lists TCP/UDP ports the agent is blocked from reaching.
	// Only applicable for network-egress programs.
	// +optional
	DeniedPorts []int32 `json:"deniedPorts,omitempty"`

	// DeniedSyscalls lists system calls the agent container is blocked from making.
	// Only applicable for syscall-filter programs.
	// +optional
	DeniedSyscalls []string `json:"deniedSyscalls,omitempty"`
}

// OPARuleConfig holds configuration for an OPA (Open Policy Agent) rule.
type OPARuleConfig struct {
	// Rego is the inline Rego policy code.
	// +kubebuilder:validation:MinLength=1
	Rego string `json:"rego"`

	// DataConfigMapRef references a ConfigMap containing data for the OPA policy.
	// +optional
	DataConfigMapRef *corev1.ObjectReference `json:"dataConfigMapRef,omitempty"`
}

// PolicyRule defines a single rule within a policy.
type PolicyRule struct {
	// Name is a human-readable identifier for this rule.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Action defines what happens when this rule matches.
	// +kubebuilder:default=block
	Action PolicyAction `json:"action,omitempty"`

	// EBPF holds configuration when this is an eBPF-based rule.
	// +optional
	EBPF *EBPFRuleConfig `json:"ebpf,omitempty"`

	// OPA holds configuration when this is an OPA-based rule.
	// +optional
	OPA *OPARuleConfig `json:"opa,omitempty"`
}

// PolicySpec defines the desired state of Policy.
type PolicySpec struct {
	// Type specifies the enforcement engine: eBPF, OPA, or hybrid (both).
	Type PolicyType `json:"type"`

	// Enforcement defines whether the policy actively blocks or only observes.
	// +kubebuilder:default=enforcing
	Enforcement EnforcementMode `json:"enforcement,omitempty"`

	// Description provides a human-readable description of the policy's purpose.
	// +optional
	Description string `json:"description,omitempty"`

	// TargetRef references the Agent or ToolServer this policy applies to.
	// If Namespace is not specified, defaults to the same namespace as the Policy.
	// +optional
	TargetRef *corev1.ObjectReference `json:"targetRef,omitempty"`

	// Rules defines the individual policy rules.
	// +kubebuilder:validation:MinItems=1
	Rules []PolicyRule `json:"rules"`
}

// PolicyStatus defines the observed state of Policy.
type PolicyStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Enforced indicates whether the policy is currently being enforced.
	// +optional
	Enforced bool `json:"enforced,omitempty"`

	// RuleCount is the number of rules in this policy.
	// +optional
	RuleCount int32 `json:"ruleCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Enforcement",type=string,JSONPath=".spec.enforcement"
// +kubebuilder:printcolumn:name="Rules",type=integer,JSONPath=".status.ruleCount"

// Policy is the Schema for the policies API.
// It defines security and cost policies enforced via eBPF and OPA.
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicySpec   `json:"spec,omitempty"`
	Status PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PolicyList contains a list of Policy.
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Policy{}, &PolicyList{})
}
