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

// Package observability provides Prometheus metrics for the Agent Runtime Operator.
// All metrics are registered automatically when this package is imported.
// Metrics cover: reconciliation, cost, tokens, policy, WASM sandbox, Merkle verification, and swarm coordination.
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// --- Agent Reconciliation Metrics ---

	// AgentReconcileTotal counts total reconciliation attempts per agent.
	AgentReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_agent_reconcile_total",
			Help: "Total number of agent reconciliation attempts",
		},
		[]string{"name", "namespace", "result"},
	)

	// AgentReconcileDuration tracks reconciliation duration in seconds.
	AgentReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agentk_agent_reconcile_duration_seconds",
			Help:    "Duration of agent reconciliation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"name", "namespace"},
	)

	// AgentReadyGauge shows which agents are currently ready (1) or not (0).
	AgentReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_agent_ready",
			Help: "Whether the agent is ready (1) or not (0)",
		},
		[]string{"name", "namespace", "framework"},
	)

	// --- Cost & Token Metrics ---

	// TokensUsedTotal tracks total tokens consumed per agent.
	TokensUsedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_tokens_used_total",
			Help: "Total tokens consumed by agent (from external metrics)",
		},
		[]string{"name", "namespace"},
	)

	// PredictedMonthlyCostUSD tracks predicted monthly cost per agent.
	PredictedMonthlyCostUSD = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_predicted_monthly_cost_usd",
			Help: "Predicted monthly cost in USD for the agent",
		},
		[]string{"name", "namespace"},
	)

	// CostActionTotal counts cost actions taken (none, downgraded, paused).
	CostActionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_cost_action_total",
			Help: "Total cost actions taken per agent",
		},
		[]string{"name", "namespace", "action"},
	)

	// --- Policy Enforcement Metrics ---

	// PolicyViolationsTotal counts policy violations detected.
	PolicyViolationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_policy_violations_total",
			Help: "Total policy violations detected",
		},
		[]string{"policy", "namespace", "rule", "action"},
	)

	// PolicyReconcileTotal counts policy reconciliation attempts.
	PolicyReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_policy_reconcile_total",
			Help: "Total policy reconciliation attempts",
		},
		[]string{"name", "namespace", "type", "result"},
	)

	// PolicyEnforcedGauge shows active enforced policies.
	PolicyEnforcedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_policy_enforced",
			Help: "Whether the policy is actively enforced (1) or not (0)",
		},
		[]string{"name", "namespace", "type"},
	)

	// --- WASM Sandbox Metrics ---

	// WASMSandboxReadyGauge shows which sandboxes are ready.
	WASMSandboxReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_wasm_sandbox_ready",
			Help: "Whether the WASM sandbox is ready (1) or not (0)",
		},
		[]string{"name", "namespace", "runtime"},
	)

	// WASMSandboxReconcileTotal counts sandbox reconciliation attempts.
	WASMSandboxReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_wasm_sandbox_reconcile_total",
			Help: "Total WASM sandbox reconciliation attempts",
		},
		[]string{"name", "namespace", "result"},
	)

	// --- Merkle Verification Metrics ---

	// MerkleCheckpointTotal counts Merkle checkpoints created.
	MerkleCheckpointTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_merkle_checkpoint_total",
			Help: "Total Merkle checkpoints created per agent",
		},
		[]string{"name", "namespace"},
	)

	// MerkleCheckpointCountGauge tracks current checkpoint count per agent.
	MerkleCheckpointCountGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_merkle_checkpoint_count",
			Help: "Current Merkle checkpoint count per agent",
		},
		[]string{"name", "namespace"},
	)

	// --- Swarm Coordination Metrics ---

	// SwarmReadyAgentsGauge tracks ready agents per swarm.
	SwarmReadyAgentsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_swarm_ready_agents",
			Help: "Number of ready agents in the swarm",
		},
		[]string{"name", "namespace", "strategy"},
	)

	// SwarmTotalAgentsGauge tracks total agents per swarm.
	SwarmTotalAgentsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_swarm_total_agents",
			Help: "Total number of agents in the swarm",
		},
		[]string{"name", "namespace", "strategy"},
	)

	// SwarmReconcileTotal counts swarm reconciliation attempts.
	SwarmReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_swarm_reconcile_total",
			Help: "Total swarm reconciliation attempts",
		},
		[]string{"name", "namespace", "strategy", "result"},
	)

	// --- SimulationPreview Metrics ---

	// SimulationPreviewTotal counts simulation previews generated.
	SimulationPreviewTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_simulation_preview_total",
			Help: "Total simulation previews generated",
		},
		[]string{"name", "namespace", "result"},
	)

	// --- Sovereign: Verifiable Execution Metrics ---

	// ZKProofChainLength tracks the proof chain length per agent.
	ZKProofChainLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_zk_proof_chain_length",
			Help: "Length of the zk-proof chain per agent",
		},
		[]string{"name", "namespace"},
	)

	// AttestationTotal counts attestation reports generated.
	AttestationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_attestation_total",
			Help: "Total attestation reports generated per agent",
		},
		[]string{"name", "namespace", "proof_mode"},
	)

	// --- Sovereign: Governance Metrics ---

	// GovernanceComplianceGauge shows governance compliance (1=compliant, 0=non-compliant).
	GovernanceComplianceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_governance_compliant",
			Help: "Whether the agent is governance-compliant (1) or not (0)",
		},
		[]string{"name", "namespace"},
	)

	// GovernanceAutonomyLevel tracks the configured autonomy level per agent.
	GovernanceAutonomyLevel = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_governance_autonomy_level",
			Help: "Configured autonomy level (1-5) per agent",
		},
		[]string{"name", "namespace"},
	)

	// --- Sovereign: Cost Intelligence Metrics ---

	// CostOptimizationActionsTotal counts real-time cost optimization actions taken.
	CostOptimizationActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_cost_optimization_actions_total",
			Help: "Total cost optimization actions taken per agent",
		},
		[]string{"name", "namespace", "mode"},
	)

	// SpotInstanceGauge shows whether spot instance scheduling is enabled (1) or not (0).
	SpotInstanceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_spot_instance_enabled",
			Help: "Whether spot instance fallback is enabled (1) or not (0)",
		},
		[]string{"name", "namespace"},
	)

	// --- Sovereign: Confidential Execution Metrics ---

	// TEEAgentReadyGauge shows which confidential agents are ready.
	TEEAgentReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_tee_agent_ready",
			Help: "Whether the TEE confidential agent is ready (1) or not (0)",
		},
		[]string{"name", "namespace", "tee_provider"},
	)

	// TEEAttestationTotal counts TEE attestation reports generated.
	TEEAttestationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentk_tee_attestation_total",
			Help: "Total TEE attestation reports generated",
		},
		[]string{"name", "namespace", "tee_provider"},
	)

	// --- Workforce Discovery Metrics ---

	// WorkforceTransitiveAgentsGauge tracks discovered transitive agents.
	WorkforceTransitiveAgentsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_workforce_transitive_agents",
			Help: "Number of transitive agents discovered in the workforce",
		},
		[]string{"name", "namespace"},
	)

	// WorkforceTransitiveToolsGauge tracks discovered transitive tools.
	WorkforceTransitiveToolsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentk_workforce_transitive_tools",
			Help: "Number of transitive tools discovered in the workforce",
		},
		[]string{"name", "namespace"},
	)
)

func init() {
	// Register all metrics with the controller-runtime metrics registry.
	// This makes them available on the /metrics endpoint automatically.
	metrics.Registry.MustRegister(
		// Agent reconciliation
		AgentReconcileTotal,
		AgentReconcileDuration,
		AgentReadyGauge,

		// Cost & tokens
		TokensUsedTotal,
		PredictedMonthlyCostUSD,
		CostActionTotal,

		// Policy enforcement
		PolicyViolationsTotal,
		PolicyReconcileTotal,
		PolicyEnforcedGauge,

		// WASM sandbox
		WASMSandboxReadyGauge,
		WASMSandboxReconcileTotal,

		// Merkle verification
		MerkleCheckpointTotal,
		MerkleCheckpointCountGauge,

		// Swarm coordination
		SwarmReadyAgentsGauge,
		SwarmTotalAgentsGauge,
		SwarmReconcileTotal,

		// Simulation preview
		SimulationPreviewTotal,

		// Sovereign - Verifiable
		ZKProofChainLength,
		AttestationTotal,

		// Sovereign - Governance
		GovernanceComplianceGauge,
		GovernanceAutonomyLevel,

		// Sovereign - Cost Intelligence
		CostOptimizationActionsTotal,
		SpotInstanceGauge,

		// Sovereign - Confidential
		TEEAgentReadyGauge,
		TEEAttestationTotal,

		// Workforce discovery
		WorkforceTransitiveAgentsGauge,
		WorkforceTransitiveToolsGauge,
	)
}
