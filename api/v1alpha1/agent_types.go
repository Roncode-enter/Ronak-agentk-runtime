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

const (
	A2AProtocol = "A2A"
)

// AgentProtocol defines a port configuration for the agent
type AgentProtocol struct {
	// Name is the name of the port
	// +kubebuilder:default=``
	Name string `json:"name,omitempty"`

	// Type of the protocol used by the agent
	// +kubebuilder:validation:Enum=A2A
	Type string `json:"type"`

	// Port is the port number, defaults to the default port for the protocol
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	// Path is the path used for HTTP-based protocols
	// +kubebuilder:validation:Pattern=`^/[a-zA-Z0-9/_-]*$`
	Path string `json:"path,omitempty"`
}

// SubAgent defines configuration for connecting to either a cluster agent or remote agent
type SubAgent struct {
	// Name is a descriptive identifier for this sub-agent connection
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// AgentRef references an Agent resource in the cluster.
	// When specified, the operator will resolve the Agent's service URL automatically.
	// Only Name and Namespace fields are used; other fields (Kind, APIVersion, etc.) are ignored.
	// If Namespace is not specified, defaults to the same namespace as the current Agent.
	// Mutually exclusive with Url - exactly one must be specified.
	// +optional
	AgentRef *corev1.ObjectReference `json:"agentRef,omitempty"`

	// Url is the HTTP/HTTPS endpoint URL for a remote agent outside the cluster.
	// It refers to the agent's well-known agent card URL, e.g. https://agent.example.com/.well-known/agent-card.json
	// Mutually exclusive with AgentRef - exactly one must be specified.
	// +optional
	// +kubebuilder:validation:Format=uri
	Url string `json:"url,omitempty"`

	// InteractionType specifies how the agent should interact with this sub-agent.
	// +optional
	// +kubebuilder:validation:Enum=transfer;tool_call
	// +kubebuilder:default="tool_call"
	InteractionType string `json:"interactionType,omitempty"`
}

// AgentTool defines configuration for integrating an MCP (Model Context Protocol) tool
type AgentTool struct {
	// Name is the unique identifier for this tool
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// ToolServerRef references a ToolServer resource in the cluster.
	// The operator will resolve the ToolServer's service URL automatically.
	// Only Name and Namespace fields are used; other fields (Kind, APIVersion, etc.) are ignored.
	// If Namespace is not specified, defaults to the same namespace as the current Agent.
	// Mutually exclusive with Url - exactly one must be specified.
	// +optional
	ToolServerRef *corev1.ObjectReference `json:"toolServerRef,omitempty"`

	// Url is the HTTP/HTTPS endpoint URL for an MCP tool server outside the cluster.
	// Mutually exclusive with ToolServerRef - exactly one must be specified.
	// +optional
	// +kubebuilder:validation:Format=uri
	Url string `json:"url,omitempty"`

	// PropagatedHeaders is a list of HTTP header names that should be propagated from incoming
	// A2A requests to this MCP tool server. This enables authentication and authorization
	// scenarios where the MCP server needs to access external APIs on behalf of users.
	// Header names are case-insensitive. If not specified or empty, no headers are propagated.
	// +optional
	PropagatedHeaders []string `json:"propagatedHeaders,omitempty"`
}

// CostBudgetSpec defines cost budget configuration for predictive cost management.
// When configured, the operator projects monthly cost from token usage and can
// automatically downgrade the model or pause the agent to stay within budget.
type CostBudgetSpec struct {
	// MaxMonthlyCostUSD is the maximum monthly cost threshold in USD (e.g., "50.00").
	// When predicted cost exceeds this, the operator takes action (downgrade or pause).
	// +kubebuilder:validation:Pattern=`^\d+(\.\d+)?$`
	MaxMonthlyCostUSD string `json:"maxMonthlyCostUSD"`

	// DowngradeModel is the cheaper model to switch to if predicted cost exceeds the budget.
	// If not set and budget is exceeded, the agent will be paused instead.
	// +optional
	DowngradeModel string `json:"downgradeModel,omitempty"`

	// CostPerTokenUSD is the cost per token in USD (e.g., "0.00001").
	// Used to calculate predicted monthly cost from token usage rate.
	// +kubebuilder:validation:Pattern=`^\d+(\.\d+)?$`
	CostPerTokenUSD string `json:"costPerTokenUSD"`
}

// CostIntelligenceSpec configures real-time cost optimization beyond basic CostBudget.
// It adds optimization modes, spot-instance scheduling hints, and budget-exhaust suspension.
type CostIntelligenceSpec struct {
	// OptimizationMode controls how aggressively the optimizer acts on cost predictions.
	// conservative: downgrade at 90% of budget, aggressive: at 70%, auto: at 80%.
	// +kubebuilder:validation:Enum=conservative;aggressive;auto
	// +kubebuilder:default="conservative"
	OptimizationMode string `json:"optimizationMode,omitempty"`

	// SpotInstanceFallback adds scheduler annotations for spot/preemptible node scheduling.
	// +optional
	SpotInstanceFallback bool `json:"spotInstanceFallback,omitempty"`

	// SuspendOnBudgetExhaust scales replicas to 0 when the budget is fully exhausted.
	// +optional
	SuspendOnBudgetExhaust bool `json:"suspendOnBudgetExhaust,omitempty"`

	// SamplingIntervalSeconds controls how often cost is re-evaluated during reconcile.
	// +optional
	// +kubebuilder:default=60
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=3600
	SamplingIntervalSeconds int32 `json:"samplingIntervalSeconds,omitempty"`
}

// VerifiableSpec configures verifiable execution with cryptographic proof chains.
// Proof chains build on the existing Merkle checkpoint system with additional attestation.
type VerifiableSpec struct {
	// Enabled turns on verifiable execution proof generation.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ProofMode selects the proof algorithm.
	// merkle-only: SHA-256 Merkle chain (free tier),
	// sha3-attestation: SHA-256 proof chain + signed attestation (legacy),
	// full-zk: alias for snark-groth16 (legacy),
	// snark-groth16: real Groth16 zk-SNARK proof via gnark (standard tier),
	// plonk-universal: PlonK universal SNARK via gnark (premium tier).
	// +kubebuilder:validation:Enum=merkle-only;sha3-attestation;full-zk;snark-groth16;plonk-universal
	// +kubebuilder:default="merkle-only"
	ProofMode string `json:"proofMode,omitempty"`

	// AttestationSignerSecret references a Kubernetes Secret containing a signing key for attestation reports.
	// The Secret should have a key named "signing-key".
	// +optional
	AttestationSignerSecret string `json:"attestationSignerSecret,omitempty"`

	// ProofRetentionDays controls how long proof data is retained in status annotations.
	// +optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=365
	ProofRetentionDays int32 `json:"proofRetentionDays,omitempty"`
}

// GovernanceSpec configures the centralized governance layer with autonomy tiers
// and human-in-loop approval gates.
type GovernanceSpec struct {
	// AutonomyLevel defines the agent's autonomy tier from 1 (human approves everything)
	// to 5 (fully autonomous). Levels 1-2 require human approval, 3 requires policy compliance,
	// 4-5 are advisory only.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:default=3
	AutonomyLevel int32 `json:"autonomyLevel,omitempty"`

	// HumanApprovalWebhook is the URL called for human-in-loop approval at low autonomy levels (1-2).
	// +optional
	// +kubebuilder:validation:Format=uri
	HumanApprovalWebhook string `json:"humanApprovalWebhook,omitempty"`

	// RequirePolicyCompliance ensures all referenced policies must pass before deployment proceeds.
	// +optional
	// +kubebuilder:default=true
	RequirePolicyCompliance bool `json:"requirePolicyCompliance,omitempty"`
}

// LifecycleSpec configures Kubernetes-style lifecycle control for the agent,
// including deployment strategies, prompt versioning, and graceful shutdown.
type LifecycleSpec struct {
	// Strategy controls how updates are rolled out.
	// rolling: standard rolling update, canary: maxSurge=1/maxUnavailable=0, blue-green: recreate.
	// +kubebuilder:validation:Enum=rolling;canary;blue-green
	// +kubebuilder:default="rolling"
	Strategy string `json:"strategy,omitempty"`

	// PromptVersion is an opaque version tag for tracking prompt/instruction changes.
	// Changes to this field trigger a new rollout and Merkle checkpoint.
	// +optional
	PromptVersion string `json:"promptVersion,omitempty"`

	// CheckpointOnUpdate forces a Merkle checkpoint before applying any spec change.
	// +optional
	// +kubebuilder:default=true
	CheckpointOnUpdate bool `json:"checkpointOnUpdate,omitempty"`

	// GracefulShutdownSeconds is the termination grace period for the agent pod.
	// +optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=600
	GracefulShutdownSeconds int32 `json:"gracefulShutdownSeconds,omitempty"`

	// SelfHealing enables automatic restart on failure via liveness probe.
	// +optional
	// +kubebuilder:default=true
	SelfHealing bool `json:"selfHealing,omitempty"`

	// MaxSurge for rolling/canary updates (percentage string like "25%" or absolute like "1").
	// +optional
	// +kubebuilder:default="25%"
	MaxSurge string `json:"maxSurge,omitempty"`

	// MaxUnavailable for rolling updates (percentage string like "25%" or absolute like "0").
	// +optional
	// +kubebuilder:default="25%"
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
}

// AgentSpec defines the desired state of Agent.
type AgentSpec struct {
	// Framework defines the supported agent frameworks
	// +kubebuilder:validation:Enum=google-adk;msaf;custom
	// +optional
	Framework string `json:"framework,omitempty"`

	// Replicas is the number of replicas for the microservice deployment
	// +kubebuilder:validation:Minimum=0
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Image is the Docker image and tag to use for the microservice deployment.
	// When not specified, the operator will use a framework-specific template image.
	// +optional
	Image string `json:"image,omitempty"`

	// Description provides a description of the agent.
	// This is passed as AGENT_DESCRIPTION environment variable to the agent.
	// +optional
	Description string `json:"description,omitempty"`

	// Instruction defines the system instruction/prompt for the agent when using template images.
	// This is passed as AGENT_INSTRUCTION environment variable to the agent.
	// +optional
	Instruction string `json:"instruction,omitempty"`

	// Model specifies the language model to use for the agent.
	// This is passed as AGENT_MODEL environment variable to the agent.
	// Defaults to the agents default model if not specified.
	// +optional
	Model string `json:"model,omitempty"`

	// SubAgents defines configuration for connecting to cluster or remote agents.
	// This is converted to JSON and passed as SUB_AGENTS environment variable to the agent.
	// +optional
	SubAgents []SubAgent `json:"subAgents,omitempty"`

	// Tools defines configuration for integrating MCP (Model Context Protocol) tools.
	// This is converted to JSON and passed as AGENT_TOOLS environment variable to the agent.
	// +optional
	Tools []AgentTool `json:"tools,omitempty"`

	// Protocols defines the protocols supported by the agent
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=atomic
	Protocols []AgentProtocol `json:"protocols,omitempty"`

	// Exposed indicates whether this agent should be exposed via the AgentGateway
	// +kubebuilder:default=false
	Exposed bool `json:"exposed,omitempty"`

	// AiGatewayRef references an AiGateway resource that this agent should use for model routing.
	// If not specified, the operator will attempt to find the default AiGateway in the cluster.
	// If no default AiGateway exists, the agent will run without an AI Gateway.
	// If Namespace is not specified, defaults to the same namespace as the Agent.
	// +optional
	AiGatewayRef *corev1.ObjectReference `json:"aiGatewayRef,omitempty"`

	// Env defines additional environment variables to be injected into the agent container.
	// These are take precedence over operator-managed environment variables.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom defines sources to populate environment variables from.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// VolumeMounts defines volume mounts to be added to the agent container.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes defines volumes to be added to the agent pod.
	// Volume names starting with "agent-operator-" are reserved for operator use and will be rejected.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Resources defines the compute resource requirements for the agent container.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ToolSandboxRef references a ToolSandbox for WASM-based secure tool execution.
	// When set, the operator injects a WasmEdge sidecar container into the agent pod.
	// If Namespace is not specified, defaults to the same namespace as the Agent.
	// +optional
	ToolSandboxRef *corev1.ObjectReference `json:"toolSandboxRef,omitempty"`

	// PolicyRefs references Policy resources to enforce on this agent.
	// When set, the operator injects eBPF/OPA enforcement sidecars into the agent pod.
	// +optional
	PolicyRefs []corev1.ObjectReference `json:"policyRefs,omitempty"`

	// CostBudget defines optional cost budget configuration for predictive cost management.
	// When set, the operator monitors token usage and predicts monthly cost.
	// If predicted cost exceeds the budget, the operator can downgrade the model or pause the agent.
	// +optional
	CostBudget *CostBudgetSpec `json:"costBudget,omitempty"`

	// CostIntelligence configures real-time cost optimization beyond basic CostBudget.
	// Adds optimization modes (conservative/aggressive/auto), spot-instance hints, and budget-exhaust suspension.
	// +optional
	CostIntelligence *CostIntelligenceSpec `json:"costIntelligence,omitempty"`

	// Verifiable configures verifiable execution with cryptographic proof chains and attestation reports.
	// +optional
	Verifiable *VerifiableSpec `json:"verifiable,omitempty"`

	// Governance configures the centralized governance layer with autonomy tiers and human-in-loop gates.
	// +optional
	Governance *GovernanceSpec `json:"governance,omitempty"`

	// Lifecycle configures Kubernetes-style lifecycle control (rolling, canary, blue-green, checkpoint).
	// +optional
	Lifecycle *LifecycleSpec `json:"lifecycle,omitempty"`

	// CommonMetadata defines labels and annotations to be applied to the Deployment and Service
	// resources created for this agent, as well as the pod template.
	// +optional
	CommonMetadata *EmbeddedMetadata `json:"commonMetadata,omitempty"`

	// PodMetadata defines labels and annotations to be applied only to the pod template
	// of the Deployment created for this agent.
	// +optional
	PodMetadata *EmbeddedMetadata `json:"podMetadata,omitempty"`
}

// AgentStatus defines the observed state of Agent.
type AgentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Url is the cluster-local URL where this agent can be accessed via A2A protocol.
	// This is automatically populated by the controller when the agent has an A2A protocol configured.
	// Format: http://{name}.{namespace}.svc.cluster.local:{port}/.well-known/agent-card.json
	// +optional
	Url string `json:"url,omitempty"`

	// AiGatewayRef references the AiGateway resource that this agent is connected to.
	// This field is automatically populated by the controller when an AI Gateway is being used.
	// If nil, the agent is not connected to any AI Gateway.
	// +optional
	AiGatewayRef *corev1.ObjectReference `json:"aiGatewayRef,omitempty"`

	// MerkleRoot is the current Merkle root hash of all reconciliation checkpoints.
	// This provides an immutable audit trail — if any checkpoint is tampered with, the root won't match.
	// Automatically populated by the controller on every reconciliation.
	// +optional
	MerkleRoot string `json:"merkleRoot,omitempty"`

	// CheckpointCount tracks the total number of reconciliation checkpoints created.
	// +optional
	CheckpointCount int32 `json:"checkpointCount,omitempty"`

	// LastCheckpointTime is the timestamp of the most recent reconciliation checkpoint.
	// +optional
	LastCheckpointTime *metav1.Time `json:"lastCheckpointTime,omitempty"`

	// PredictedMonthlyCostUSD is the estimated monthly cost in USD based on current token usage rate.
	// Only populated when CostBudget is configured on the agent spec.
	// +optional
	PredictedMonthlyCostUSD string `json:"predictedMonthlyCostUSD,omitempty"`

	// CostAction indicates what action was taken based on cost prediction.
	// Possible values: "none" (within budget), "downgraded" (switched to cheaper model), "paused" (over budget, no downgrade available).
	// +optional
	CostAction string `json:"costAction,omitempty"`

	// TokensUsed tracks the total tokens consumed by this agent.
	// This field is intended to be updated by external metrics systems (e.g., Prometheus, agent runtime).
	// The controller reads this value to project costs but does not write it.
	// +optional
	TokensUsed int64 `json:"tokensUsed,omitempty"`

	// ZKProofRoot is the root hash of the verifiable proof chain (SHA-256, zk-SNARK ready).
	// Populated when spec.verifiable.enabled is true.
	// +optional
	ZKProofRoot string `json:"zkProofRoot,omitempty"`

	// AttestationDigest is the latest signed attestation report hash.
	// Provides a tamper-proof digest that can be verified externally.
	// +optional
	AttestationDigest string `json:"attestationDigest,omitempty"`

	// GovernanceStatus shows the current governance compliance state.
	// Possible values: "compliant", "pending-approval", "non-compliant".
	// +optional
	GovernanceStatus string `json:"governanceStatus,omitempty"`

	// LifecyclePhase shows the current lifecycle phase of the agent.
	// Possible values: "stable", "updating", "suspended".
	// +optional
	LifecyclePhase string `json:"lifecyclePhase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AI Gateway",type=string,JSONPath=".status.aiGatewayRef.name"
// +kubebuilder:printcolumn:name="Merkle Root",type=string,JSONPath=".status.merkleRoot",priority=1
// +kubebuilder:printcolumn:name="Cost",type=string,JSONPath=".status.predictedMonthlyCostUSD"
// +kubebuilder:printcolumn:name="Cost Action",type=string,JSONPath=".status.costAction"
// +kubebuilder:printcolumn:name="Governance",type=string,JSONPath=".status.governanceStatus",priority=1
// +kubebuilder:printcolumn:name="Lifecycle",type=string,JSONPath=".status.lifecyclePhase",priority=1

// Agent is the Schema for the agents API.
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentList contains a list of Agent.
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}
