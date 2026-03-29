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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	"github.com/agentic-layer/agent-runtime-operator/internal/swarm"
)

var swarmlog = logf.Log.WithName("swarm-resource")

// SetupSwarmWebhookWithManager registers the webhook for Swarm in the manager.
func SetupSwarmWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &runtimev1alpha1.Swarm{}).
		WithDefaulter(&SwarmCustomDefaulter{}).
		WithValidator(&SwarmCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-runtime-agentic-layer-ai-v1alpha1-swarm,mutating=true,failurePolicy=fail,sideEffects=None,groups=runtime.agentic-layer.ai,resources=swarms,verbs=create;update,versions=v1alpha1,name=mswarm-v1alpha1.kb.io,admissionReviewVersions=v1

// SwarmCustomDefaulter struct is responsible for setting default values on the Swarm resource.
//
// +kubebuilder:object:generate=false
type SwarmCustomDefaulter struct{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Swarm.
func (d *SwarmCustomDefaulter) Default(_ context.Context, s *runtimev1alpha1.Swarm) error {
	swarmlog.Info("Defaulting for Swarm", "name", s.GetName())

	// Default coordinatorImage to the operator's built-in default
	if s.Spec.CoordinatorImage == "" {
		s.Spec.CoordinatorImage = swarm.DefaultCoordinatorImage
		swarmlog.Info("Defaulted coordinatorImage", "image", swarm.DefaultCoordinatorImage)
	}

	// Default replicas to 1
	if s.Spec.Replicas == nil {
		replicas := int32(1)
		s.Spec.Replicas = &replicas
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-runtime-agentic-layer-ai-v1alpha1-swarm,mutating=false,failurePolicy=fail,sideEffects=None,groups=runtime.agentic-layer.ai,resources=swarms,verbs=create;update,versions=v1alpha1,name=vswarm-v1alpha1.kb.io,admissionReviewVersions=v1

// SwarmCustomValidator struct validates the Swarm resource.
//
// +kubebuilder:object:generate=false
type SwarmCustomValidator struct{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the Kind Swarm.
func (v *SwarmCustomValidator) ValidateCreate(_ context.Context, s *runtimev1alpha1.Swarm) (admission.Warnings, error) {
	swarmlog.Info("Validating Swarm create", "name", s.GetName())
	return v.validateSwarm(s)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the Kind Swarm.
func (v *SwarmCustomValidator) ValidateUpdate(_ context.Context, _ *runtimev1alpha1.Swarm, s *runtimev1alpha1.Swarm) (admission.Warnings, error) {
	swarmlog.Info("Validating Swarm update", "name", s.GetName())
	return v.validateSwarm(s)
}

// ValidateDelete implements webhook.CustomValidator.
func (v *SwarmCustomValidator) ValidateDelete(_ context.Context, _ *runtimev1alpha1.Swarm) (admission.Warnings, error) {
	return nil, nil
}

// validateSwarm performs validation on the Swarm spec.
func (v *SwarmCustomValidator) validateSwarm(s *runtimev1alpha1.Swarm) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	agentsPath := field.NewPath("spec", "agents")

	// Validate each agent has exactly one of agentRef or url
	for i, agent := range s.Spec.Agents {
		agentPath := agentsPath.Index(i)
		hasRef := agent.AgentRef != nil
		hasUrl := agent.Url != ""

		if !hasRef && !hasUrl {
			allErrs = append(allErrs, field.Required(agentPath,
				fmt.Sprintf("agent %q must specify either agentRef or url", agent.Name)))
		}
		if hasRef && hasUrl {
			allErrs = append(allErrs, field.Invalid(agentPath.Child("url"), agent.Url,
				fmt.Sprintf("agent %q must specify either agentRef or url, not both", agent.Name)))
		}
	}

	// Validate leader-follower strategy has at least one leader
	if s.Spec.Strategy == runtimev1alpha1.SwarmStrategyLeaderFollower {
		hasLeader := false
		for _, agent := range s.Spec.Agents {
			if agent.Role == "leader" {
				hasLeader = true
				break
			}
		}
		if !hasLeader {
			allErrs = append(allErrs, field.Required(agentsPath,
				"leader-follower strategy requires at least one agent with role \"leader\""))
		}
	}

	// Warn if coordinatorImage is the default (which may not exist in the registry)
	if s.Spec.CoordinatorImage == swarm.DefaultCoordinatorImage {
		warnings = append(warnings,
			fmt.Sprintf("spec.coordinatorImage is set to the default image %q which may not exist in your container registry. "+
				"Set spec.coordinatorImage to a valid image to avoid ImagePullBackOff errors.", swarm.DefaultCoordinatorImage))
	}

	if len(allErrs) > 0 {
		return warnings, allErrs.ToAggregate()
	}

	return warnings, nil
}
