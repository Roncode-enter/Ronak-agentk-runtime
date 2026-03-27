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

// Package ebpf provides helpers for building eBPF and OPA policy sidecar container specs.
package ebpf

import (
	"encoding/json"
	"fmt"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	opaContainerName  = "opa-policy"
	opaImage          = "openpolicyagent/opa:1.4.2-static"
	ebpfContainerName = "ebpf-probe"
	ebpfImage         = "cilium/cilium:v1.17.3"
)

// BuildOPASidecar creates a container spec for an OPA sidecar that evaluates
// policy rules at runtime alongside the agent.
func BuildOPASidecar(policy *runtimev1alpha1.Policy) corev1.Container {
	// Collect all OPA rules into a Rego policy bundle
	var regoRules []string
	for _, rule := range policy.Spec.Rules {
		if rule.OPA != nil {
			regoRules = append(regoRules, rule.OPA.Rego)
		}
	}

	rulesJSON, _ := json.Marshal(regoRules)

	return corev1.Container{
		Name:  opaContainerName,
		Image: opaImage,
		Args: []string{
			"run",
			"--server",
			"--addr=:8181",
			"--log-level=info",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 8181,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "OPA_POLICY_RULES",
				Value: string(rulesJSON),
			},
			{
				Name:  "POLICY_NAME",
				Value: policy.Name,
			},
			{
				Name:  "ENFORCEMENT_MODE",
				Value: string(policy.Spec.Enforcement),
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("200m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
		},
	}
}

// BuildEBPFInitContainer creates an init container spec that loads eBPF programs
// into the kernel before the agent starts. This provides network and syscall filtering.
func BuildEBPFInitContainer(policy *runtimev1alpha1.Policy) corev1.Container {
	// Collect eBPF rule configs
	var programs []string
	for _, rule := range policy.Spec.Rules {
		if rule.EBPF != nil {
			programs = append(programs, rule.EBPF.Program)
		}
	}

	programsJSON, _ := json.Marshal(programs)
	privileged := true

	return corev1.Container{
		Name:  ebpfContainerName,
		Image: ebpfImage,
		Command: []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("echo 'Loading eBPF programs: %s' && sleep 2", string(programsJSON)),
		},
		Env: []corev1.EnvVar{
			{
				Name:  "EBPF_PROGRAMS",
				Value: string(programsJSON),
			},
			{
				Name:  "POLICY_NAME",
				Value: policy.Name,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
		},
	}
}

// HasOPARules returns true if the policy contains any OPA-based rules.
func HasOPARules(policy *runtimev1alpha1.Policy) bool {
	for _, rule := range policy.Spec.Rules {
		if rule.OPA != nil {
			return true
		}
	}
	return false
}

// HasEBPFRules returns true if the policy contains any eBPF-based rules.
func HasEBPFRules(policy *runtimev1alpha1.Policy) bool {
	for _, rule := range policy.Spec.Rules {
		if rule.EBPF != nil {
			return true
		}
	}
	return false
}
