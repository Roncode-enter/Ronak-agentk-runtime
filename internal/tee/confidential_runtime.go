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

// Package tee provides builder functions for Trusted Execution Environment (TEE) configuration.
// It builds Kata Confidential Container specs, attestation sidecars, and security contexts
// for deploying agents inside hardware-rooted enclaves (Intel TDX, AMD SEV-SNP, AWS Nitro).
package tee

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
)

const (
	// AttestationSidecarName is the container name for the attestation agent sidecar.
	AttestationSidecarName = "attestation-agent"
	// AttestationImage is the container image for the attestation agent.
	AttestationImage = "ghcr.io/agentic-layer/attestation-agent:0.1.0"
	// DefaultRuntimeClassName is the default Kata Confidential Containers runtime class.
	DefaultRuntimeClassName = "kata-cc"
	// AttestationPort is the port the attestation sidecar listens on.
	AttestationPort int32 = 9090
)

// ConfidentialPodConfig holds the TEE-specific pod configuration.
type ConfidentialPodConfig struct {
	// RuntimeClassName for Kata Confidential Containers.
	RuntimeClassName *string
	// Annotations for TEE metadata on the pod.
	Annotations map[string]string
	// EnclaveMemoryMB is the enclave memory allocation.
	EnclaveMemoryMB int32
}

// BuildRuntimeClassName returns the runtime class name from spec or the default.
func BuildRuntimeClassName(spec *runtimev1alpha1.ConfidentialAgentSpec) string {
	if spec.RuntimeClassName != "" {
		return spec.RuntimeClassName
	}
	return DefaultRuntimeClassName
}

// BuildSecurityContext returns a hardened security context for TEE containers.
func BuildSecurityContext() *corev1.SecurityContext {
	nonRoot := true
	readOnly := true
	return &corev1.SecurityContext{
		RunAsNonRoot:             &nonRoot,
		ReadOnlyRootFilesystem:   &readOnly,
		AllowPrivilegeEscalation: new(bool), // false
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

// BuildAttestationSidecar creates the attestation agent sidecar container.
// This sidecar periodically generates TEE attestation reports and exposes them on port 9090.
func BuildAttestationSidecar(spec *runtimev1alpha1.ConfidentialAgentSpec) corev1.Container {
	memEncryption := "false"
	if spec.MemoryEncryption {
		memEncryption = "true"
	}

	intervalSeconds := spec.AttestationIntervalSeconds
	if intervalSeconds == 0 {
		intervalSeconds = 300
	}

	return corev1.Container{
		Name:  AttestationSidecarName,
		Image: AttestationImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          "attestation",
				ContainerPort: AttestationPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{Name: "TEE_PROVIDER", Value: string(spec.Provider)},
			{Name: "ATTESTATION_ENDPOINT", Value: spec.AttestationEndpoint},
			{Name: "ATTESTATION_INTERVAL_SECONDS", Value: fmt.Sprintf("%d", intervalSeconds)},
			{Name: "MEMORY_ENCRYPTION", Value: memEncryption},
			{Name: "ENCLAVE_MEMORY_MB", Value: fmt.Sprintf("%d", spec.EnclaveMemoryMB)},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(AttestationPort),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
		SecurityContext: BuildSecurityContext(),
	}
}

// BuildConfidentialPodConfig returns TEE-specific pod configuration.
func BuildConfidentialPodConfig(spec *runtimev1alpha1.ConfidentialAgentSpec) *ConfidentialPodConfig {
	runtimeClass := BuildRuntimeClassName(spec)
	enclaveMemMB := spec.EnclaveMemoryMB
	if enclaveMemMB == 0 {
		enclaveMemMB = 256
	}

	annotations := map[string]string{
		"agentk.io/tee-provider":       string(spec.Provider),
		"agentk.io/runtime-class":      runtimeClass,
		"agentk.io/memory-encryption":  fmt.Sprintf("%t", spec.MemoryEncryption),
		"agentk.io/enclave-memory-mb":  fmt.Sprintf("%d", enclaveMemMB),
		"io.katacontainers.config.hypervisor.memory_encryption": fmt.Sprintf("%t", spec.MemoryEncryption),
	}

	return &ConfidentialPodConfig{
		RuntimeClassName: &runtimeClass,
		Annotations:      annotations,
		EnclaveMemoryMB:  enclaveMemMB,
	}
}
