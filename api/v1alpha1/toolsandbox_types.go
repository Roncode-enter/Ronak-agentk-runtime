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

// SandboxRuntime defines the WASM runtime used for the sandbox.
// +kubebuilder:validation:Enum=wasmedge
type SandboxRuntime string

const (
	SandboxRuntimeWasmEdge SandboxRuntime = "wasmedge"
)

// ToolSandboxSpec defines the desired state of ToolSandbox.
type ToolSandboxSpec struct {
	// Runtime specifies the WASM runtime to use for sandboxed execution.
	// +kubebuilder:default=wasmedge
	Runtime SandboxRuntime `json:"runtime,omitempty"`

	// Image is the OCI image containing the WASM module to execute.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// MemoryLimitMB defines the maximum memory in megabytes available to the WASM module.
	// +optional
	// +kubebuilder:default=64
	// +kubebuilder:validation:Minimum=16
	// +kubebuilder:validation:Maximum=1024
	MemoryLimitMB int32 `json:"memoryLimitMB,omitempty"`

	// TimeoutSeconds defines the maximum execution time for a single WASM invocation.
	// +optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// AllowedHosts lists the hostnames or IPs the WASM module is allowed to connect to.
	// An empty list means no outbound network access.
	// +optional
	// +listType=set
	AllowedHosts []string `json:"allowedHosts,omitempty"`

	// Port specifies the port the sandbox HTTP server listens on.
	// +optional
	// +kubebuilder:default=8080
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	// Replicas is the number of sandbox instances to run.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// Env defines additional environment variables for the sandbox container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// CommonMetadata defines labels and annotations to be applied to the
	// Deployment and Service resources created for this sandbox.
	// +optional
	CommonMetadata *EmbeddedMetadata `json:"commonMetadata,omitempty"`
}

// ToolSandboxStatus defines the observed state of ToolSandbox.
type ToolSandboxStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Url is the cluster-internal URL for reaching this sandbox.
	// +optional
	Url string `json:"url,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=".spec.runtime"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=".spec.image"

// ToolSandbox is the Schema for the toolsandboxes API.
// It defines a WASM-based sandbox for secure, isolated tool execution.
type ToolSandbox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolSandboxSpec   `json:"spec,omitempty"`
	Status ToolSandboxStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ToolSandboxList contains a list of ToolSandbox.
type ToolSandboxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolSandbox `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ToolSandbox{}, &ToolSandboxList{})
}
