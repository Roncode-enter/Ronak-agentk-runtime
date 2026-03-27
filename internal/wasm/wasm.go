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

// Package wasm provides helpers for building WasmEdge sidecar container specs.
package wasm

import (
	"fmt"
	"strings"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	sidecarContainerName = "wasm-sandbox"
	wasmEdgeImage        = "wasmedge/wasmedge:0.14.1"
)

// BuildSidecarContainer creates a container spec for a WasmEdge sidecar
// that runs alongside the agent to provide sandboxed tool execution.
func BuildSidecarContainer(sandbox *runtimev1alpha1.ToolSandbox) corev1.Container {
	memoryLimit := fmt.Sprintf("%dMi", sandbox.Spec.MemoryLimitMB)

	env := make([]corev1.EnvVar, 0, len(sandbox.Spec.Env)+3)
	env = append(env, corev1.EnvVar{
		Name:  "WASM_MODULE_IMAGE",
		Value: sandbox.Spec.Image,
	})
	env = append(env, corev1.EnvVar{
		Name:  "WASM_TIMEOUT_SECONDS",
		Value: fmt.Sprintf("%d", sandbox.Spec.TimeoutSeconds),
	})
	if len(sandbox.Spec.AllowedHosts) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "WASM_ALLOWED_HOSTS",
			Value: strings.Join(sandbox.Spec.AllowedHosts, ","),
		})
	}
	env = append(env, sandbox.Spec.Env...)

	return corev1.Container{
		Name:  sidecarContainerName,
		Image: wasmEdgeImage,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: sandbox.Spec.Port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: env,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(memoryLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intOrString(sandbox.Spec.Port),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}
}

func intOrString(port int32) intstr.IntOrString {
	return intstr.FromInt32(port)
}
