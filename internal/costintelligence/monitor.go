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

package costintelligence

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
)

// CostMonitor is a background goroutine that continuously evaluates agent costs.
// It implements controller-runtime's manager.Runnable interface so the manager
// starts and stops it alongside the controllers.
type CostMonitor struct {
	Client   client.Client
	Log      logr.Logger
	Interval time.Duration

	cache sync.Map // map[types.NamespacedName]*CostDecision
}

// NewCostMonitor creates a new CostMonitor with the given sampling interval.
func NewCostMonitor(c client.Client, log logr.Logger, interval time.Duration) *CostMonitor {
	if interval < 10*time.Second {
		interval = 60 * time.Second
	}
	return &CostMonitor{
		Client:   c,
		Log:      log.WithName("cost-monitor"),
		Interval: interval,
	}
}

// Start begins the continuous cost evaluation loop. Blocks until ctx is cancelled.
// This satisfies the manager.Runnable interface.
func (m *CostMonitor) Start(ctx context.Context) error {
	m.Log.Info("Real-time cost monitor started", "interval", m.Interval.String())
	ticker := time.NewTicker(m.Interval)
	defer ticker.Stop()

	// Run immediately on start
	m.evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			m.Log.Info("Cost monitor stopped")
			return nil
		case <-ticker.C:
			m.evaluate(ctx)
		}
	}
}

// GetDecision returns the latest cached cost decision for an agent, or nil if not available.
func (m *CostMonitor) GetDecision(key types.NamespacedName) *CostDecision {
	val, ok := m.cache.Load(key)
	if !ok {
		return nil
	}
	return val.(*CostDecision)
}

// evaluate scans all agents with cost budgets and computes fresh cost decisions.
func (m *CostMonitor) evaluate(ctx context.Context) {
	var agentList runtimev1alpha1.AgentList
	if err := m.Client.List(ctx, &agentList); err != nil {
		m.Log.Error(err, "Failed to list agents for cost evaluation")
		return
	}

	evaluated := 0
	for i := range agentList.Items {
		agent := &agentList.Items[i]
		if agent.Spec.CostBudget == nil {
			continue
		}

		key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}

		// Read actual token usage from pod annotations.
		// Agent runtimes report token counts via the annotation "agentk.io/tokens-used"
		// on their pods. This is the primary mechanism for real token tracking.
		tokensUsed := agent.Status.TokensUsed
		if tokensUsed == 0 {
			tokensUsed = m.readTokensFromPods(ctx, agent)
		}

		// Parse cost parameters
		costPerToken, _ := strconv.ParseFloat(agent.Spec.CostBudget.CostPerTokenUSD, 64)
		maxMonthlyCost, _ := strconv.ParseFloat(agent.Spec.CostBudget.MaxMonthlyCostUSD, 64)
		uptimeSeconds := time.Since(agent.CreationTimestamp.Time).Seconds()

		// Get optimization mode
		optimizationMode := "auto"
		suspendOnExhaust := false
		spotFallback := false
		if agent.Spec.CostIntelligence != nil {
			if agent.Spec.CostIntelligence.OptimizationMode != "" {
				optimizationMode = agent.Spec.CostIntelligence.OptimizationMode
			}
			suspendOnExhaust = agent.Spec.CostIntelligence.SuspendOnBudgetExhaust
			spotFallback = agent.Spec.CostIntelligence.SpotInstanceFallback
		}

		decision := EvaluateRealTimeCost(
			tokensUsed,
			uptimeSeconds,
			costPerToken,
			maxMonthlyCost,
			agent.Spec.CostBudget.DowngradeModel,
			optimizationMode,
			suspendOnExhaust,
			spotFallback,
		)

		m.cache.Store(key, decision)
		evaluated++
	}

	if evaluated > 0 {
		m.Log.V(1).Info("Cost evaluation complete", "agentsEvaluated", evaluated)
	}
}

// readTokensFromPods reads token usage from agent pod annotations.
// Agent runtimes are expected to report token counts via the annotation "agentk.io/tokens-used".
// This aggregates token counts across all pods matching the agent's app label.
func (m *CostMonitor) readTokensFromPods(ctx context.Context, agent *runtimev1alpha1.Agent) int64 {
	var podList corev1.PodList
	labelSelector := labels.SelectorFromSet(labels.Set{"app": agent.Name})
	if err := m.Client.List(ctx, &podList,
		client.InNamespace(agent.Namespace),
		client.MatchingLabelsSelector{Selector: labelSelector},
	); err != nil {
		m.Log.V(2).Info("Failed to list pods for token counting", "agent", agent.Name, "error", err)
		return 0
	}

	var totalTokens int64
	for _, pod := range podList.Items {
		if val, ok := pod.Annotations["agentk.io/tokens-used"]; ok {
			tokens, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				totalTokens += tokens
			}
		}
	}

	return totalTokens
}
