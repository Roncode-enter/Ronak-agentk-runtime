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

// Package lifecycle provides Kubernetes-style lifecycle control for agents.
// It maps agent lifecycle strategies to native Kubernetes deployment strategies
// and configures graceful shutdown, self-healing, and prompt versioning.
package lifecycle

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ConfigureDeploymentStrategy maps agent lifecycle strategy to a Kubernetes DeploymentStrategy.
//
// These strategies use native Kubernetes Deployment primitives:
//
//   - "rolling": Standard RollingUpdate with configurable maxSurge/maxUnavailable.
//     This is the default strategy and provides zero-downtime rolling updates.
//
//   - "canary": Implemented as RollingUpdate with maxSurge=1 and maxUnavailable=0.
//     This introduces exactly one new pod at a time while keeping all existing pods running,
//     providing a canary-style gradual rollout using Kubernetes-native mechanisms.
//     NOTE: This is NOT a traffic-splitting canary (which would require a service mesh like
//     Istio or Linkerd). It is a pod-level rollout strategy. For traffic-percentage-based
//     canary deployments, use a service mesh with the Agent's Deployment directly.
//
//   - "blue-green": Implemented as Recreate strategy, which terminates all old pods before
//     creating new ones. This provides a clean cut-over between versions.
//     NOTE: This is NOT a true blue-green deployment (which maintains two full environments
//     and switches traffic atomically). True blue-green requires two Deployments and a Service
//     selector switch, which the operator may support in a future release. The current
//     implementation ensures version isolation (no mixed versions running simultaneously)
//     at the cost of brief downtime during the switchover.
func ConfigureDeploymentStrategy(strategy string, maxSurge string, maxUnavailable string) appsv1.DeploymentStrategy {
	switch strategy {
	case "canary":
		surge := intstr.FromInt32(1)
		unavailable := intstr.FromInt32(0)
		return appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxSurge:       &surge,
				MaxUnavailable: &unavailable,
			},
		}
	case "blue-green":
		return appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	default: // "rolling"
		surge := intstr.FromString(maxSurge)
		unavailable := intstr.FromString(maxUnavailable)
		return appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxSurge:       &surge,
				MaxUnavailable: &unavailable,
			},
		}
	}
}

// BuildLifecycleAnnotations returns pod annotations for lifecycle metadata.
// Includes the user-facing strategy name and the actual Kubernetes implementation used.
func BuildLifecycleAnnotations(strategy string, promptVersion string, checkpointOnUpdate bool) map[string]string {
	// Map user-facing strategy to the actual Kubernetes implementation
	k8sStrategy := "RollingUpdate"
	switch strategy {
	case "canary":
		k8sStrategy = "RollingUpdate (maxSurge=1, maxUnavailable=0)"
	case "blue-green":
		k8sStrategy = "Recreate"
	}

	annotations := map[string]string{
		"agentk.io/lifecycle-strategy":    strategy,
		"agentk.io/k8s-strategy-impl":    k8sStrategy,
		"agentk.io/checkpoint-on-update":  fmt.Sprintf("%t", checkpointOnUpdate),
	}
	if promptVersion != "" {
		annotations["agentk.io/prompt-version"] = promptVersion
	}
	return annotations
}

// DetermineLifecyclePhase determines the current lifecycle phase based on agent state.
func DetermineLifecyclePhase(replicas *int32, observedGeneration int64, currentGeneration int64) string {
	// Suspended: replicas explicitly set to 0
	if replicas != nil && *replicas == 0 {
		return "suspended"
	}
	// Updating: generation mismatch means a change is being rolled out
	if observedGeneration > 0 && observedGeneration != currentGeneration {
		return "updating"
	}
	return "stable"
}

// ConfigureGracefulShutdown sets the termination grace period on the pod spec.
func ConfigureGracefulShutdown(podSpec *corev1.PodSpec, seconds int32) {
	s := int64(seconds)
	podSpec.TerminationGracePeriodSeconds = &s
}

// ConfigureSelfHealing adds a liveness probe to the container for automatic restart on failure.
func ConfigureSelfHealing(container *corev1.Container, port int32) {
	if container.LivenessProbe != nil {
		return // Don't overwrite existing probe
	}
	container.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt32(port),
			},
		},
		InitialDelaySeconds: 15,
		PeriodSeconds:       20,
		TimeoutSeconds:      3,
		FailureThreshold:    3,
	}
}
