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

// SwarmStrategy defines how agents in a Swarm coordinate with each other.
// +kubebuilder:validation:Enum=round-robin;fan-out;chain;leader-follower
type SwarmStrategy string

const (
	// SwarmStrategyRoundRobin distributes incoming requests evenly across agents.
	SwarmStrategyRoundRobin SwarmStrategy = "round-robin"

	// SwarmStrategyFanOut sends each request to all agents and aggregates responses.
	SwarmStrategyFanOut SwarmStrategy = "fan-out"

	// SwarmStrategyChain passes requests through agents sequentially in a pipeline.
	SwarmStrategyChain SwarmStrategy = "chain"

	// SwarmStrategyLeaderFollower has one leader agent that delegates to follower agents.
	SwarmStrategyLeaderFollower SwarmStrategy = "leader-follower"
)

// SwarmAgent represents an agent participating in a Swarm.
type SwarmAgent struct {
	// Name is a unique identifier for this agent within the swarm.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// AgentRef references an Agent resource in the cluster.
	// When specified, the operator will resolve the Agent's service URL automatically.
	// Only Name and Namespace fields are used; other fields are ignored.
	// If Namespace is not specified, defaults to the same namespace as the Swarm.
	// Mutually exclusive with Url - exactly one must be specified.
	// +optional
	AgentRef *corev1.ObjectReference `json:"agentRef,omitempty"`

	// Url is the HTTP/HTTPS endpoint URL for a remote agent outside the cluster.
	// It refers to the agent's well-known agent card URL.
	// Mutually exclusive with AgentRef - exactly one must be specified.
	// +optional
	// +kubebuilder:validation:Format=uri
	Url string `json:"url,omitempty"`

	// Role defines this agent's role in the swarm coordination.
	// Only meaningful for leader-follower and chain strategies.
	// +optional
	// +kubebuilder:validation:Enum=leader;worker;aggregator
	Role string `json:"role,omitempty"`
}

// SwarmSpec defines the desired state of Swarm.
type SwarmSpec struct {
	// Strategy defines how agents in this swarm coordinate with each other.
	// +kubebuilder:validation:Required
	Strategy SwarmStrategy `json:"strategy"`

	// Agents defines the agents participating in this swarm.
	// At least 2 agents are required for meaningful coordination.
	// +kubebuilder:validation:MinItems=2
	Agents []SwarmAgent `json:"agents"`

	// MaxConcurrency is the maximum number of parallel agent calls for fan-out strategy.
	// Only used when strategy is "fan-out". Defaults to the number of agents.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxConcurrency *int32 `json:"maxConcurrency,omitempty"`

	// TimeoutSeconds is the per-agent call timeout in seconds.
	// +optional
	// +kubebuilder:default=60
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// CoordinatorImage is the container image for the swarm coordinator pod.
	// The coordinator routes A2A requests between agents according to the swarm strategy.
	// If not set, defaults to the operator's built-in default image.
	// +optional
	CoordinatorImage string `json:"coordinatorImage,omitempty"`

	// CoordinatorPort is the port the coordinator service listens on.
	// +optional
	// +kubebuilder:default=8080
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	CoordinatorPort int32 `json:"coordinatorPort,omitempty"`

	// Replicas is the number of coordinator pod replicas.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// CommonMetadata defines labels and annotations to be applied to resources
	// created for this swarm (Deployment, Service, ConfigMap).
	// +optional
	CommonMetadata *EmbeddedMetadata `json:"commonMetadata,omitempty"`
}

// SwarmStatus defines the observed state of Swarm.
type SwarmStatus struct {
	// Conditions represent the latest available observations of the swarm's state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// ReadyAgents is the number of agents that are reachable and have a valid URL.
	// +optional
	ReadyAgents int32 `json:"readyAgents,omitempty"`

	// TotalAgents is the total number of agents defined in the swarm.
	// +optional
	TotalAgents int32 `json:"totalAgents,omitempty"`

	// CoordinatorUrl is the cluster-local URL of the coordinator service.
	// Format: http://{name}-coordinator.{namespace}.svc.cluster.local:{port}
	// +optional
	CoordinatorUrl string `json:"coordinatorUrl,omitempty"`

	// Strategy echoes the active coordination strategy from the spec.
	// +optional
	Strategy string `json:"strategy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Strategy",type=string,JSONPath=".spec.strategy"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=".status.readyAgents"
// +kubebuilder:printcolumn:name="Total",type=integer,JSONPath=".status.totalAgents"
// +kubebuilder:printcolumn:name="Coordinator URL",type=string,JSONPath=".status.coordinatorUrl",priority=1

// Swarm is the Schema for the swarms API.
// A Swarm defines active multi-agent coordination — it deploys a coordinator pod
// that routes A2A requests between participating agents using a defined strategy.
type Swarm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwarmSpec   `json:"spec,omitempty"`
	Status SwarmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SwarmList contains a list of Swarm.
type SwarmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Swarm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Swarm{}, &SwarmList{})
}
