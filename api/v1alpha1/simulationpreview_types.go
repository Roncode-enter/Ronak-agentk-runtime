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

// SimulationPreviewSpec defines the desired state of SimulationPreview.
// A SimulationPreview lets you preview what an Agent deployment would look like
// without actually creating it — like a "dry run" for your agent.
type SimulationPreviewSpec struct {
	// AgentRef references the Agent resource to preview.
	// Only Name and Namespace fields are used.
	// If Namespace is not specified, defaults to the same namespace as the SimulationPreview.
	// +kubebuilder:validation:Required
	AgentRef corev1.ObjectReference `json:"agentRef"`

	// DryRun indicates this is a preview-only operation. Always true.
	// +kubebuilder:default=true
	DryRun bool `json:"dryRun,omitempty"`
}

// SimulationPreviewStatus defines the observed state of SimulationPreview.
type SimulationPreviewStatus struct {
	// Conditions represent the latest available observations of the SimulationPreview's state.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// PreviewDeployment contains the YAML representation of what the Deployment would look like.
	// +optional
	PreviewDeployment string `json:"previewDeployment,omitempty"`

	// EstimatedCost contains the estimated monthly cost if the agent has CostBudget configured.
	// +optional
	EstimatedCost string `json:"estimatedCost,omitempty"`

	// Warnings lists any configuration issues found during preview generation.
	// +optional
	Warnings []string `json:"warnings,omitempty"`

	// ContainerCount is the total number of containers (main + sidecars) in the previewed pod.
	// +optional
	ContainerCount int32 `json:"containerCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=".spec.agentRef.name"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Containers",type=integer,JSONPath=".status.containerCount"
// +kubebuilder:printcolumn:name="Est. Cost",type=string,JSONPath=".status.estimatedCost"

// SimulationPreview is the Schema for the simulationpreviews API.
// It provides a dry-run preview of what an Agent deployment would look like.
type SimulationPreview struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimulationPreviewSpec   `json:"spec,omitempty"`
	Status SimulationPreviewStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SimulationPreviewList contains a list of SimulationPreview.
type SimulationPreviewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimulationPreview `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SimulationPreview{}, &SimulationPreviewList{})
}
