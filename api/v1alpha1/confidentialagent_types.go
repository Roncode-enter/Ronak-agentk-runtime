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

// TEEProvider defines the Trusted Execution Environment provider.
// +kubebuilder:validation:Enum=intel-tdx;amd-sev-snp;aws-nitro;kata-cc
type TEEProvider string

const (
	// TEEIntelTDX uses Intel Trust Domain Extensions for hardware-level isolation.
	TEEIntelTDX TEEProvider = "intel-tdx"

	// TEEAMDSevSNP uses AMD Secure Encrypted Virtualization with Secure Nested Paging.
	TEEAMDSevSNP TEEProvider = "amd-sev-snp"

	// TEEAWSNitro uses AWS Nitro Enclaves for cloud-native confidential computing.
	TEEAWSNitro TEEProvider = "aws-nitro"

	// TEEKataCC uses Kata Confidential Containers for generic TEE deployment.
	TEEKataCC TEEProvider = "kata-cc"
)

// ConfidentialAgentSpec defines the desired state of ConfidentialAgent.
// A ConfidentialAgent wraps an existing Agent in a hardware-rooted Trusted Execution Environment (TEE).
type ConfidentialAgentSpec struct {
	// AgentRef references the Agent resource to deploy inside the TEE.
	// Only Name and Namespace fields are used.
	// If Namespace is not specified, defaults to the same namespace as the ConfidentialAgent.
	// +kubebuilder:validation:Required
	AgentRef corev1.ObjectReference `json:"agentRef"`

	// TEEProvider specifies the hardware TEE provider for confidential execution.
	// +kubebuilder:validation:Required
	Provider TEEProvider `json:"teeProvider"`

	// RuntimeClassName is the Kubernetes RuntimeClass to use for TEE pods.
	// +optional
	// +kubebuilder:default="kata-cc"
	RuntimeClassName string `json:"runtimeClassName,omitempty"`

	// MemoryEncryption enables hardware memory encryption for the agent pod.
	// +optional
	// +kubebuilder:default=true
	MemoryEncryption bool `json:"memoryEncryption,omitempty"`

	// AttestationEndpoint is the URL for remote attestation verification.
	// When set, the attestation sidecar periodically reports to this endpoint.
	// +optional
	// +kubebuilder:validation:Format=uri
	AttestationEndpoint string `json:"attestationEndpoint,omitempty"`

	// EnclaveMemoryMB is the memory allocated to the TEE enclave in megabytes.
	// +optional
	// +kubebuilder:default=256
	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=16384
	EnclaveMemoryMB int32 `json:"enclaveMemoryMB,omitempty"`

	// AttestationIntervalSeconds is how often the attestation sidecar generates a fresh report.
	// +optional
	// +kubebuilder:default=300
	// +kubebuilder:validation:Minimum=30
	// +kubebuilder:validation:Maximum=3600
	AttestationIntervalSeconds int32 `json:"attestationIntervalSeconds,omitempty"`
}

// ConfidentialAgentStatus defines the observed state of ConfidentialAgent.
type ConfidentialAgentStatus struct {
	// Conditions represent the latest available observations of the ConfidentialAgent's state.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// AttestationReport contains the latest TEE attestation report hash.
	// +optional
	AttestationReport string `json:"attestationReport,omitempty"`

	// TEEProvider echoes the active TEE provider from the spec.
	// +optional
	TEEProvider string `json:"teeProvider,omitempty"`

	// Verified indicates whether the latest attestation was successful.
	// +optional
	Verified bool `json:"verified,omitempty"`

	// LastAttestationTime is the timestamp of the most recent attestation report.
	// +optional
	LastAttestationTime *metav1.Time `json:"lastAttestationTime,omitempty"`

	// DeploymentName is the name of the TEE-enabled Deployment created for this ConfidentialAgent.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TEE Provider",type=string,JSONPath=".spec.teeProvider"
// +kubebuilder:printcolumn:name="Verified",type=boolean,JSONPath=".status.verified"
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=".spec.agentRef.name"
// +kubebuilder:printcolumn:name="Last Attestation",type=date,JSONPath=".status.lastAttestationTime",priority=1

// ConfidentialAgent is the Schema for the confidentialagents API.
// It deploys an existing Agent inside a hardware-rooted Trusted Execution Environment (TEE)
// with remote attestation, memory encryption, and per-step zk-proofs.
type ConfidentialAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfidentialAgentSpec   `json:"spec,omitempty"`
	Status ConfidentialAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConfidentialAgentList contains a list of ConfidentialAgent.
type ConfidentialAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfidentialAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfidentialAgent{}, &ConfidentialAgentList{})
}
