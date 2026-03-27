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

// Package costpredictor provides linear-regression-based cost prediction for agent workloads.
// It projects current token usage rate to estimate monthly cost, and recommends actions
// (downgrade model or pause) when the predicted cost exceeds the configured budget.
package costpredictor

import (
	"fmt"
	"math"
	"strconv"
)

const (
	// secondsInMonth is the number of seconds in a 30-day month (30 * 24 * 60 * 60).
	secondsInMonth = 2592000

	// ActionNone means predicted cost is within budget — no action needed.
	ActionNone = "none"

	// ActionDowngraded means predicted cost exceeds budget and a cheaper model is available.
	ActionDowngraded = "downgraded"

	// ActionPaused means predicted cost exceeds budget and no downgrade model is configured.
	ActionPaused = "paused"
)

// CostPrediction holds the result of a cost prediction calculation.
type CostPrediction struct {
	// PredictedMonthlyCostUSD is the estimated monthly cost formatted as a string (e.g. "42.50").
	PredictedMonthlyCostUSD string

	// Action indicates what action was taken: "none", "downgraded", or "paused".
	Action string

	// DowngradeModel is the cheaper model to switch to, populated only when Action is "downgraded".
	DowngradeModel string
}

// PredictCost uses linear extrapolation to estimate monthly cost from current token usage.
// It calculates the token consumption rate (tokens/second), projects it to a full 30-day month,
// and multiplies by the cost per token to get the predicted monthly cost in USD.
//
// If the predicted cost exceeds maxMonthlyCostUSD:
//   - If downgradeModel is provided, Action is set to "downgraded"
//   - Otherwise, Action is set to "paused"
//
// If uptimeSeconds is zero or tokensUsed is zero, the predicted cost is $0.00.
func PredictCost(tokensUsed int64, uptimeSeconds float64, costPerTokenUSD float64,
	maxMonthlyCostUSD float64, downgradeModel string) CostPrediction {

	monthlyCost := linearProjectMonthly(tokensUsed, uptimeSeconds) * costPerTokenUSD

	// Round to 2 decimal places
	monthlyCost = math.Round(monthlyCost*100) / 100

	prediction := CostPrediction{
		PredictedMonthlyCostUSD: fmt.Sprintf("%.2f", monthlyCost),
		Action:                  ActionNone,
	}

	// Check if over budget
	if monthlyCost > maxMonthlyCostUSD && maxMonthlyCostUSD > 0 {
		if downgradeModel != "" {
			prediction.Action = ActionDowngraded
			prediction.DowngradeModel = downgradeModel
		} else {
			prediction.Action = ActionPaused
		}
	}

	return prediction
}

// ParseCostString converts a string like "50.00" or "0.00001" to a float64.
// Returns an error if the string is not a valid number.
func ParseCostString(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty cost string")
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid cost string %q: %w", s, err)
	}
	if val < 0 {
		return 0, fmt.Errorf("negative cost value: %s", s)
	}
	return val, nil
}

// linearProjectMonthly extrapolates current token usage to a full 30-day month.
// Formula: rate = tokensUsed / uptimeSeconds, projected = rate * secondsInMonth.
func linearProjectMonthly(tokensUsed int64, uptimeSeconds float64) float64 {
	if uptimeSeconds <= 0 || tokensUsed <= 0 {
		return 0
	}
	rate := float64(tokensUsed) / uptimeSeconds
	return rate * secondsInMonth
}
