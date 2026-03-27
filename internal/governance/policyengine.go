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

// Package governance provides centralized policy enforcement and autonomy tier evaluation.
// It evaluates governance rules based on autonomy level (1-5) and policy compliance,
// determining whether an agent deployment should proceed, wait for human approval, or be blocked.
package governance

import (
	"fmt"
)

// GovernanceDecision represents the result of governance evaluation.
type GovernanceDecision struct {
	// Status is the governance compliance state: "compliant", "pending-approval", "non-compliant".
	Status string
	// Allowed indicates whether the deployment should proceed.
	Allowed bool
	// RequiresHumanApproval is true when autonomy level requires human sign-off.
	RequiresHumanApproval bool
	// Reason provides a human-readable explanation of the decision.
	Reason string
}

const (
	StatusCompliant       = "compliant"
	StatusPendingApproval = "pending-approval"
	StatusNonCompliant    = "non-compliant"
)

// EvaluateGovernance evaluates the governance rules for an agent deployment.
//
// Autonomy levels:
//   - Level 1-2: Requires human approval + all policies must be compliant
//   - Level 3: Requires policy compliance, no human gate
//   - Level 4-5: Advisory only — deployment always proceeds
func EvaluateGovernance(
	autonomyLevel int32,
	policyRefCount int,
	policyCompliant bool,
	requireCompliance bool,
) *GovernanceDecision {
	decision := &GovernanceDecision{
		Allowed: true,
	}

	// Level 4-5: Fully autonomous — always compliant
	if autonomyLevel >= 4 {
		decision.Status = StatusCompliant
		decision.Reason = fmt.Sprintf("Autonomy level %d: fully autonomous, advisory policies only", autonomyLevel)
		return decision
	}

	// Level 3: Requires policy compliance if enabled
	if autonomyLevel == 3 {
		if requireCompliance && policyRefCount > 0 && !policyCompliant {
			decision.Status = StatusNonCompliant
			decision.Allowed = false
			decision.Reason = "Autonomy level 3: policy compliance required but policies are not compliant"
			return decision
		}
		decision.Status = StatusCompliant
		decision.Reason = "Autonomy level 3: policies compliant"
		return decision
	}

	// Level 1-2: Requires human approval + policy compliance
	if requireCompliance && policyRefCount > 0 && !policyCompliant {
		decision.Status = StatusNonCompliant
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("Autonomy level %d: policy compliance required but policies are not compliant", autonomyLevel)
		return decision
	}

	decision.Status = StatusPendingApproval
	decision.RequiresHumanApproval = true
	decision.Reason = fmt.Sprintf("Autonomy level %d: human approval required", autonomyLevel)
	return decision
}

// BuildGovernanceAnnotations returns pod annotations for governance metadata.
func BuildGovernanceAnnotations(autonomyLevel int32, status string, humanWebhook string) map[string]string {
	annotations := map[string]string{
		"agentk.io/autonomy-level":    fmt.Sprintf("%d", autonomyLevel),
		"agentk.io/governance-status": status,
	}
	if humanWebhook != "" {
		annotations["agentk.io/human-approval-webhook"] = humanWebhook
	}
	return annotations
}
