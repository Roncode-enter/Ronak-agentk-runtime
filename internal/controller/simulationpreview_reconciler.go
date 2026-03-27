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
	"github.com/agentic-layer/agent-runtime-operator/internal/costpredictor"
	"github.com/agentic-layer/agent-runtime-operator/internal/observability"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// SimulationPreviewReconciler reconciles a SimulationPreview object
type SimulationPreviewReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=simulationpreviews,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=simulationpreviews/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=simulationpreviews/finalizers,verbs=update

// Reconcile generates a preview of what an Agent deployment would look like without creating it.
func (r *SimulationPreviewReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the SimulationPreview instance
	var preview runtimev1alpha1.SimulationPreview
	if err := r.Get(ctx, req.NamespacedName, &preview); err != nil {
		if errors.IsNotFound(err) {
			log.Info("SimulationPreview resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get SimulationPreview")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling SimulationPreview")

	// Resolve the referenced Agent
	agentNs := preview.Spec.AgentRef.Namespace
	if agentNs == "" {
		agentNs = preview.Namespace
	}

	var agent runtimev1alpha1.Agent
	if err := r.Get(ctx, types.NamespacedName{
		Name:      preview.Spec.AgentRef.Name,
		Namespace: agentNs,
	}, &agent); err != nil {
		if errors.IsNotFound(err) {
			if statusErr := r.updateStatusNotReady(ctx, &preview, "AgentNotFound",
				fmt.Sprintf("Agent %s/%s not found", agentNs, preview.Spec.AgentRef.Name)); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Build the preview deployment spec
	previewDeployment, warnings := r.buildPreviewDeployment(&agent)

	// Marshal to YAML
	deploymentYAML, err := yaml.Marshal(previewDeployment)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to marshal preview deployment: %w", err)
	}

	// Compute estimated cost if CostBudget is configured
	estimatedCost := ""
	if agent.Spec.CostBudget != nil {
		costPerToken, parseErr := costpredictor.ParseCostString(agent.Spec.CostBudget.CostPerTokenUSD)
		if parseErr == nil {
			// Estimate based on a typical usage of 1M tokens/month
			monthlyCost := 1000000 * costPerToken
			estimatedCost = fmt.Sprintf("~$%.2f/mo (at 1M tokens)", monthlyCost)
		}
	}

	// Count containers
	containerCount := int32(len(previewDeployment.Spec.Template.Spec.Containers))
	containerCount += int32(len(previewDeployment.Spec.Template.Spec.InitContainers))

	// Update status
	if err := r.updateStatusReady(ctx, &preview, string(deploymentYAML), estimatedCost, warnings, containerCount); err != nil {
		log.Error(err, "Failed to update SimulationPreview status")
		return ctrl.Result{}, err
	}

	// Record metrics
	observability.SimulationPreviewTotal.WithLabelValues(preview.Name, preview.Namespace, "success").Inc()

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimulationPreviewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.SimulationPreview{}).
		Named("simulationpreview").
		Complete(r)
}

// buildPreviewDeployment creates a simplified Deployment spec based on the Agent,
// mimicking what the agent controller would create.
func (r *SimulationPreviewReconciler) buildPreviewDeployment(agent *runtimev1alpha1.Agent) (*appsv1.Deployment, []string) {
	var warnings []string

	// Determine image
	image := agent.Spec.Image
	if image == "" {
		switch agent.Spec.Framework {
		case "google-adk":
			image = defaultTemplateImageAdk
		case "msaf":
			image = defaultTemplateImageMsaf
		default:
			image = defaultTemplateImageAdk
		}
		warnings = append(warnings, "No custom image specified, using template image: "+image)
	}

	// Check protocols
	if len(agent.Spec.Protocols) == 0 {
		warnings = append(warnings, "No protocols defined — agent will not have a Service")
	}

	// Build container ports
	containerPorts := make([]corev1.ContainerPort, 0, len(agent.Spec.Protocols))
	for _, protocol := range agent.Spec.Protocols {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          protocol.Name,
			ContainerPort: protocol.Port,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	// Build containers
	containers := []corev1.Container{
		{
			Name:  agentContainerName,
			Image: image,
			Ports: containerPorts,
			Env:   agent.Spec.Env,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("300Mi"),
					corev1.ResourceCPU:    resource.MustParse("100m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("500Mi"),
					corev1.ResourceCPU:    resource.MustParse("500m"),
				},
			},
		},
	}

	// Check for sidecar containers
	if agent.Spec.ToolSandboxRef != nil {
		containers = append(containers, corev1.Container{
			Name:  "wasm-sandbox",
			Image: "wasmedge/wasmedge:0.14.1",
		})
		warnings = append(warnings, fmt.Sprintf("ToolSandbox sidecar will be injected (ref: %s)", agent.Spec.ToolSandboxRef.Name))
	}

	initContainers := make([]corev1.Container, 0, len(agent.Spec.PolicyRefs))
	for _, policyRef := range agent.Spec.PolicyRefs {
		containers = append(containers, corev1.Container{
			Name:  "opa-policy",
			Image: "openpolicyagent/opa:1.4.2-static",
		})
		initContainers = append(initContainers, corev1.Container{
			Name:  "ebpf-probe",
			Image: "cilium/cilium:v1.17.3",
		})
		warnings = append(warnings, fmt.Sprintf("Policy enforcement sidecars will be injected (ref: %s)", policyRef.Name))
	}

	// Check for cost budget
	if agent.Spec.CostBudget != nil {
		warnings = append(warnings, fmt.Sprintf("Cost budget configured: max $%s/month, downgrade model: %s",
			agent.Spec.CostBudget.MaxMonthlyCostUSD, agent.Spec.CostBudget.DowngradeModel))
	}

	// Build replicas
	replicas := int32(1)
	if agent.Spec.Replicas != nil {
		replicas = *agent.Spec.Replicas
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"app": agent.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": agent.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": agent.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers:     containers,
					InitContainers: initContainers,
					Volumes:        agent.Spec.Volumes,
				},
			},
		},
	}

	return deployment, warnings
}

// updateStatusReady sets the SimulationPreview status to Ready with the preview data
func (r *SimulationPreviewReconciler) updateStatusReady(ctx context.Context, preview *runtimev1alpha1.SimulationPreview,
	previewYAML string, estimatedCost string, warnings []string, containerCount int32) error {

	preview.Status.PreviewDeployment = previewYAML
	preview.Status.EstimatedCost = estimatedCost
	preview.Status.Warnings = warnings
	preview.Status.ContainerCount = containerCount

	meta.SetStatusCondition(&preview.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "PreviewGenerated",
		Message:            "Deployment preview generated successfully",
		ObservedGeneration: preview.Generation,
	})

	if err := r.Status().Update(ctx, preview); err != nil {
		return fmt.Errorf("failed to update SimulationPreview status: %w", err)
	}

	return nil
}

// updateStatusNotReady sets the SimulationPreview status to not Ready
func (r *SimulationPreviewReconciler) updateStatusNotReady(ctx context.Context, preview *runtimev1alpha1.SimulationPreview,
	reason, message string) error {

	meta.SetStatusCondition(&preview.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: preview.Generation,
	})

	if err := r.Status().Update(ctx, preview); err != nil {
		return fmt.Errorf("failed to update SimulationPreview status: %w", err)
	}

	return nil
}
