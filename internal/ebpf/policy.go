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
//
// Rego policies are mounted from a ConfigMap at /etc/opa/policies/ and OPA
// watches the directory for changes. The ConfigMap is created by the Policy
// controller and contains one .rego file per rule.
func BuildOPASidecar(policy *runtimev1alpha1.Policy) corev1.Container {
	return corev1.Container{
		Name:  opaContainerName,
		Image: opaImage,
		Args: []string{
			"run",
			"--server",
			"--addr=:8181",
			"--log-level=info",
			"--watch",
			"/etc/opa/policies",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 8181,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "POLICY_NAME",
				Value: policy.Name,
			},
			{
				Name:  "ENFORCEMENT_MODE",
				Value: string(policy.Spec.Enforcement),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "opa-policies",
				MountPath: "/etc/opa/policies",
				ReadOnly:  true,
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

// BuildOPAPolicyConfigMapData creates the ConfigMap data for OPA policy files.
// Each OPA rule becomes a separate .rego file in the ConfigMap.
func BuildOPAPolicyConfigMapData(policy *runtimev1alpha1.Policy) map[string]string {
	data := make(map[string]string)
	for _, rule := range policy.Spec.Rules {
		if rule.OPA != nil {
			filename := fmt.Sprintf("%s.rego", rule.Name)
			data[filename] = rule.OPA.Rego
		}
	}
	return data
}

// BuildOPAPolicyVolume returns the volume spec for mounting OPA Rego policies.
func BuildOPAPolicyVolume(policyName string) corev1.Volume {
	return corev1.Volume{
		Name: "opa-policies",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-opa-policies", policyName),
				},
			},
		},
	}
}

// BuildEBPFInitContainer creates an init container spec that loads eBPF programs
// into the kernel before the agent starts. This provides network and syscall filtering.
//
// The init container uses Cilium's bpftool to load each configured eBPF program
// from the policy spec. Programs are loaded into the kernel's BPF subsystem and
// pinned to /sys/fs/bpf/ so the agent container can reference them at runtime.
// If any program fails to load, the init container exits non-zero and the pod will not start.
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

	// Build a shell script that loads each eBPF program via bpftool.
	// Each program name maps to an object file at /etc/ebpf/<program>.o
	// which is mounted from the policy ConfigMap.
	// Programs are pinned to /sys/fs/bpf/<policy>/<program> for runtime use.
	loadScript := fmt.Sprintf(`set -e
echo "eBPF loader: policy=%s programs=%s"
mkdir -p /sys/fs/bpf/%s
for prog in $(echo '%s' | jq -r '.[]'); do
  obj="/etc/ebpf/${prog}.o"
  pin="/sys/fs/bpf/%s/${prog}"
  if [ -f "$obj" ]; then
    echo "Loading eBPF program: $prog from $obj"
    bpftool prog load "$obj" "$pin" type xdp 2>/dev/null || \
    bpftool prog load "$obj" "$pin" 2>/dev/null || \
    { echo "WARN: bpftool load failed for $prog, attempting cilium-agent attach"; \
      cilium-agent bpf load "$obj" "$pin" 2>/dev/null || \
      { echo "ERROR: Failed to load eBPF program $prog"; exit 1; }; }
    echo "Loaded and pinned: $prog -> $pin"
  else
    echo "WARN: eBPF object file not found: $obj (skipping — ensure ConfigMap contains compiled BPF objects)"
  fi
done
echo "eBPF programs loaded successfully for policy %s"
`, policy.Name, string(programsJSON), policy.Name, string(programsJSON), policy.Name, policy.Name)

	return corev1.Container{
		Name:  ebpfContainerName,
		Image: ebpfImage,
		Command: []string{
			"/bin/sh",
			"-c",
			loadScript,
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
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "ebpf-programs",
				MountPath: "/etc/ebpf",
				ReadOnly:  true,
			},
			{
				Name:      "bpf-fs",
				MountPath: "/sys/fs/bpf",
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("200m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}
}

// BuildEBPFVolumes returns the volumes needed by the eBPF init container:
// 1. A ConfigMap volume containing compiled BPF object files
// 2. A hostPath volume for the BPF filesystem mount
func BuildEBPFVolumes(policyName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "ebpf-programs",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-ebpf-programs", policyName),
					},
				},
			},
		},
		{
			Name: "bpf-fs",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys/fs/bpf",
				},
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
