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

package controller

import (
	"context"
	"fmt"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	"github.com/agentic-layer/agent-runtime-operator/internal/observability"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// PolicyReconciler reconciles a Policy object
type PolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=policies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=policies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Policy instance
	var policy runtimev1alpha1.Policy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Policy resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Policy")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling Policy")

	// Validate policy rules
	if err := r.validateRules(&policy); err != nil {
		log.Error(err, "Policy validation failed")
		if statusErr := r.updateStatusNotReady(ctx, &policy, "ValidationFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status after validation failure")
		}
		return ctrl.Result{}, nil // Don't requeue, user needs to fix the policy
	}

	// Update status to Ready
	if err := r.updateStatusReady(ctx, &policy); err != nil {
		log.Error(err, "Failed to update Policy status")
		return ctrl.Result{}, err
	}

	// Record metrics
	policyType := string(policy.Spec.Type)
	observability.PolicyReconcileTotal.WithLabelValues(policy.Name, policy.Namespace, policyType, "success").Inc()
	enforced := float64(0)
	if policy.Status.Enforced {
		enforced = 1
	}
	observability.PolicyEnforcedGauge.WithLabelValues(policy.Name, policy.Namespace, policyType).Set(enforced)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.Policy{}).
		Named("policy").
		Complete(r)
}

// validateRules checks that policy rules are consistent with the policy type
func (r *PolicyReconciler) validateRules(policy *runtimev1alpha1.Policy) error {
	for _, rule := range policy.Spec.Rules {
		switch policy.Spec.Type {
		case runtimev1alpha1.PolicyTypeEBPF:
			if rule.EBPF == nil {
				return fmt.Errorf("rule %q: eBPF policy type requires ebpf configuration", rule.Name)
			}
		case runtimev1alpha1.PolicyTypeOPA:
			if rule.OPA == nil {
				return fmt.Errorf("rule %q: OPA policy type requires opa configuration", rule.Name)
			}
		case runtimev1alpha1.PolicyTypeHybrid:
			if rule.EBPF == nil && rule.OPA == nil {
				return fmt.Errorf("rule %q: hybrid policy type requires at least one of ebpf or opa configuration", rule.Name)
			}
		}
	}
	return nil
}

// updateStatusReady sets the Policy status to Ready
func (r *PolicyReconciler) updateStatusReady(ctx context.Context, policy *runtimev1alpha1.Policy) error {
	policy.Status.RuleCount = int32(len(policy.Spec.Rules))
	policy.Status.Enforced = policy.Spec.Enforcement == runtimev1alpha1.EnforcementModeEnforcing

	meta.SetStatusCondition(&policy.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Validated",
		Message:            fmt.Sprintf("Policy validated with %d rules", len(policy.Spec.Rules)),
		ObservedGeneration: policy.Generation,
	})

	if err := r.Status().Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy status: %w", err)
	}

	return nil
}

// updateStatusNotReady sets the Policy status to not Ready
func (r *PolicyReconciler) updateStatusNotReady(ctx context.Context, policy *runtimev1alpha1.Policy, reason, message string) error {
	policy.Status.Enforced = false

	meta.SetStatusCondition(&policy.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: policy.Generation,
	})

	if err := r.Status().Update(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy status: %w", err)
	}

	return nil
}
