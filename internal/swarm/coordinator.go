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

// Package swarm provides helpers for building Swarm coordinator deployment specs.
// The coordinator is a lightweight pod that routes A2A requests between participating
// agents according to the swarm's coordination strategy (round-robin, fan-out, chain, leader-follower).
package swarm

import (
	"encoding/json"
	"fmt"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// CoordinatorContainerName is the name of the coordinator container in the pod.
	CoordinatorContainerName = "swarm-coordinator"

	// CoordinatorImage is the default image for the swarm coordinator.
	CoordinatorImage = "ghcr.io/agentic-layer/swarm-coordinator:0.1.0"

	// ConfigVolumeName is the name of the ConfigMap volume mount.
	ConfigVolumeName = "swarm-config"

	// ConfigMountPath is where the config is mounted in the coordinator container.
	ConfigMountPath = "/etc/swarm"

	// ConfigFileName is the key in the ConfigMap data.
	ConfigFileName = "config.json"
)

// CoordinatorConfig is the JSON configuration passed to the coordinator container
// via a mounted ConfigMap. It tells the coordinator which agents to route to and how.
type CoordinatorConfig struct {
	Strategy       string                 `json:"strategy"`
	Agents         []CoordinatorAgentInfo `json:"agents"`
	MaxConcurrency int32                  `json:"maxConcurrency,omitempty"`
	TimeoutSeconds int32                  `json:"timeoutSeconds"`
}

// CoordinatorAgentInfo describes one agent in the coordinator's config.
type CoordinatorAgentInfo struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Role string `json:"role,omitempty"`
}

// BuildCoordinatorConfig creates the JSON configuration string for the coordinator.
// agentUrls maps agent name to its resolved A2A URL. agentRoles maps agent name to its role.
func BuildCoordinatorConfig(swarm *runtimev1alpha1.Swarm, agentUrls map[string]string) string {
	agents := make([]CoordinatorAgentInfo, 0, len(swarm.Spec.Agents))
	for _, sa := range swarm.Spec.Agents {
		url, ok := agentUrls[sa.Name]
		if !ok {
			continue
		}
		agents = append(agents, CoordinatorAgentInfo{
			Name: sa.Name,
			Url:  url,
			Role: sa.Role,
		})
	}

	timeout := int32(60)
	if swarm.Spec.TimeoutSeconds != nil {
		timeout = *swarm.Spec.TimeoutSeconds
	}

	maxConcurrency := int32(len(swarm.Spec.Agents))
	if swarm.Spec.MaxConcurrency != nil {
		maxConcurrency = *swarm.Spec.MaxConcurrency
	}

	config := CoordinatorConfig{
		Strategy:       string(swarm.Spec.Strategy),
		Agents:         agents,
		MaxConcurrency: maxConcurrency,
		TimeoutSeconds: timeout,
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

// BuildCoordinatorContainer creates the container spec for the coordinator pod.
func BuildCoordinatorContainer(swarm *runtimev1alpha1.Swarm) corev1.Container {
	port := GetCoordinatorPort(swarm)

	return corev1.Container{
		Name:  CoordinatorContainerName,
		Image: CoordinatorImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "SWARM_CONFIG_PATH",
				Value: fmt.Sprintf("%s/%s", ConfigMountPath, ConfigFileName),
			},
			{
				Name:  "SWARM_PORT",
				Value: fmt.Sprintf("%d", port),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      ConfigVolumeName,
				MountPath: ConfigMountPath,
				ReadOnly:  true,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("200m"),
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(port),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}
}

// BuildConfigVolume creates the Volume spec for the coordinator ConfigMap.
func BuildConfigVolume(configMapName string) corev1.Volume {
	return corev1.Volume{
		Name: ConfigVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

// GetCoordinatorPort returns the coordinator port, defaulting to 8080.
func GetCoordinatorPort(swarm *runtimev1alpha1.Swarm) int32 {
	if swarm.Spec.CoordinatorPort > 0 {
		return swarm.Spec.CoordinatorPort
	}
	return 8080
}

// CoordinatorName returns the name of the coordinator deployment/service.
func CoordinatorName(swarmName string) string {
	return fmt.Sprintf("%s-coordinator", swarmName)
}

// ConfigMapName returns the name of the coordinator ConfigMap.
func ConfigMapName(swarmName string) string {
	return fmt.Sprintf("%s-coordinator-config", swarmName)
}
