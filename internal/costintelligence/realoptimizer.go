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

// Package costintelligence provides real-time cost optimization logic for agents.
// It extends the basic costpredictor with optimization modes, spot-instance scheduling,
// and budget-exhaust suspension.
package costintelligence

import (
	"github.com/agentic-layer/agent-runtime-operator/internal/costpredictor"
)

// CostDecision represents the output of real-time cost evaluation.
type CostDecision struct {
	// PredictedMonthlyCostUSD is the projected monthly cost.
	PredictedMonthlyCostUSD string
	// Action taken: "none", "downgraded", "paused", "suspended".
	Action string
	// ShouldSuspend is true when budget is exhausted and suspension is configured.
	ShouldSuspend bool
	// SpotAnnotations contains pod annotations for spot/preemptible scheduling.
	SpotAnnotations map[string]string
	// DowngradeModel is the cheaper model to switch to, if applicable.
	DowngradeModel string
}

// Optimization mode thresholds (percentage of budget at which action is taken).
const (
	conservativeThreshold = 0.90
	aggressiveThreshold   = 0.70
	autoThreshold         = 0.80
)

// EvaluateRealTimeCost performs real-time cost evaluation with optimization modes.
// It wraps the basic PredictCost with threshold adjustments based on optimization mode.
func EvaluateRealTimeCost(
	tokensUsed int64,
	uptimeSeconds float64,
	costPerTokenUSD float64,
	maxMonthlyCostUSD float64,
	downgradeModel string,
	optimizationMode string,
	suspendOnExhaust bool,
	spotFallback bool,
) *CostDecision {
	// Adjust the effective budget threshold based on optimization mode
	threshold := autoThreshold
	switch optimizationMode {
	case "conservative":
		threshold = conservativeThreshold
	case "aggressive":
		threshold = aggressiveThreshold
	case "auto":
		threshold = autoThreshold
	}

	// Use the effective (reduced) budget for prediction
	effectiveBudget := maxMonthlyCostUSD * threshold

	// Call existing cost predictor with the effective budget
	prediction := costpredictor.PredictCost(tokensUsed, uptimeSeconds, costPerTokenUSD, effectiveBudget, downgradeModel)

	decision := &CostDecision{
		PredictedMonthlyCostUSD: prediction.PredictedMonthlyCostUSD,
		Action:                  prediction.Action,
		DowngradeModel:          prediction.DowngradeModel,
	}

	// Check if we should suspend (budget fully exhausted + suspend configured)
	if suspendOnExhaust && prediction.Action == costpredictor.ActionPaused {
		decision.Action = "suspended"
		decision.ShouldSuspend = true
	}

	// Add spot instance annotations if configured
	if spotFallback {
		decision.SpotAnnotations = BuildSpotAnnotations()
	}

	return decision
}

// BuildSpotAnnotations returns pod annotations for spot/preemptible node scheduling.
// These annotations are recognized by GKE, Karpenter, and other Kubernetes schedulers.
func BuildSpotAnnotations() map[string]string {
	return map[string]string{
		"cloud.google.com/gke-spot": "true",
		"karpenter.sh/capacity-type": "spot",
		"agentk.io/spot-instance":    "true",
	}
}

// ShouldSuspend returns true when the cost decision indicates the agent should be suspended.
func ShouldSuspend(decision *CostDecision) bool {
	return decision != nil && decision.ShouldSuspend
}
