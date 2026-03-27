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
//   - "rolling": RollingUpdate with configurable maxSurge/maxUnavailable
//   - "canary": RollingUpdate with maxSurge=1, maxUnavailable=0 (one new pod at a time)
//   - "blue-green": Recreate (all old pods replaced at once)
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
func BuildLifecycleAnnotations(strategy string, promptVersion string, checkpointOnUpdate bool) map[string]string {
	annotations := map[string]string{
		"agentk.io/lifecycle-strategy":   strategy,
		"agentk.io/checkpoint-on-update": fmt.Sprintf("%t", checkpointOnUpdate),
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
