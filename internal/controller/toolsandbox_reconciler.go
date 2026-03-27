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
	"maps"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	"github.com/agentic-layer/agent-runtime-operator/internal/observability"
	"github.com/agentic-layer/agent-runtime-operator/internal/wasm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	sandboxContainerName = "wasm-sandbox"
)

// ToolSandboxReconciler reconciles a ToolSandbox object
type ToolSandboxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=toolsandboxes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=toolsandboxes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=toolsandboxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ToolSandboxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the ToolSandbox instance
	var sandbox runtimev1alpha1.ToolSandbox
	if err := r.Get(ctx, req.NamespacedName, &sandbox); err != nil {
		if errors.IsNotFound(err) {
			log.Info("ToolSandbox resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ToolSandbox")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ToolSandbox")

	// Ensure Deployment exists and is up to date
	if err := r.ensureDeployment(ctx, &sandbox); err != nil {
		log.Error(err, "Failed to ensure Deployment")
		if statusErr := r.updateStatusNotReady(ctx, &sandbox, "DeploymentFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status after deployment failure")
		}
		return ctrl.Result{}, err
	}

	// Ensure Service exists and is up to date
	if err := r.ensureService(ctx, &sandbox); err != nil {
		log.Error(err, "Failed to ensure Service")
		if statusErr := r.updateStatusNotReady(ctx, &sandbox, "ServiceFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status after service failure")
		}
		return ctrl.Result{}, err
	}

	// Update status to Ready
	if err := r.updateStatusReady(ctx, &sandbox); err != nil {
		log.Error(err, "Failed to update ToolSandbox status")
		return ctrl.Result{}, err
	}

	// Record metrics
	observability.WASMSandboxReconcileTotal.WithLabelValues(sandbox.Name, sandbox.Namespace, "success").Inc()
	observability.WASMSandboxReadyGauge.WithLabelValues(sandbox.Name, sandbox.Namespace, string(sandbox.Spec.Runtime)).Set(1)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ToolSandboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.ToolSandbox{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("toolsandbox").
		Complete(r)
}

// ensureDeployment ensures the Deployment for the ToolSandbox exists and is up to date
func (r *ToolSandboxReconciler) ensureDeployment(ctx context.Context, sandbox *runtimev1alpha1.ToolSandbox) error {
	log := logf.FromContext(ctx)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sandbox.Name,
			Namespace: sandbox.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{},
				},
			},
		},
	}

	if op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		managedLabels := map[string]string{
			"app":     sandbox.Name,
			"runtime": string(sandbox.Spec.Runtime),
		}

		selectorLabels := map[string]string{
			"app": sandbox.Name,
		}

		// Set immutable fields only on creation
		if deployment.CreationTimestamp.IsZero() {
			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			}
		}

		// Build pod template labels
		podTemplateLabels, podTemplateAnnotations := buildPodTemplateMetadata(
			selectorLabels, sandbox.Spec.CommonMetadata, nil,
		)
		deployment.Spec.Template.Labels = podTemplateLabels
		deployment.Spec.Template.Annotations = podTemplateAnnotations

		// Set replicas
		if deployment.Spec.Replicas == nil {
			deployment.Spec.Replicas = new(int32)
		}
		if sandbox.Spec.Replicas != nil {
			*deployment.Spec.Replicas = *sandbox.Spec.Replicas
		} else {
			*deployment.Spec.Replicas = 1
		}

		// Merge labels
		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
		}
		applyCommonMetadataToObjectMeta(&deployment.ObjectMeta, sandbox.Spec.CommonMetadata)
		maps.Copy(deployment.Labels, managedLabels)

		// Build the WasmEdge sidecar container
		wasmContainer := wasm.BuildSidecarContainer(sandbox)

		// Update or create container
		container := findContainerByName(&deployment.Spec.Template.Spec, sandboxContainerName)
		if container == nil {
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, wasmContainer)
		} else {
			container.Image = wasmContainer.Image
			container.Ports = wasmContainer.Ports
			container.Env = wasmContainer.Env
			container.Resources = wasmContainer.Resources
			container.ReadinessProbe = wasmContainer.ReadinessProbe
		}

		// Set owner reference
		return ctrl.SetControllerReference(sandbox, deployment, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("Deployment reconciled", "operation", op)
	}

	return nil
}

// ensureService ensures the Service for the ToolSandbox exists and is up to date
func (r *ToolSandboxReconciler) ensureService(ctx context.Context, sandbox *runtimev1alpha1.ToolSandbox) error {
	log := logf.FromContext(ctx)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sandbox.Name,
			Namespace: sandbox.Namespace,
		},
	}

	if op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		managedLabels := map[string]string{
			"app":     sandbox.Name,
			"runtime": string(sandbox.Spec.Runtime),
		}

		selectorLabels := map[string]string{
			"app": sandbox.Name,
		}

		servicePorts := []corev1.ServicePort{
			{
				Name:       "sandbox",
				Port:       sandbox.Spec.Port,
				TargetPort: intstr.FromInt32(sandbox.Spec.Port),
				Protocol:   corev1.ProtocolTCP,
			},
		}

		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		applyCommonMetadataToObjectMeta(&service.ObjectMeta, sandbox.Spec.CommonMetadata)
		maps.Copy(service.Labels, managedLabels)

		service.Spec.Ports = servicePorts
		service.Spec.Selector = selectorLabels
		service.Spec.Type = corev1.ServiceTypeClusterIP

		return ctrl.SetControllerReference(sandbox, service, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("Service reconciled", "operation", op)
	}

	return nil
}

// updateStatusReady sets the ToolSandbox status to Ready
func (r *ToolSandboxReconciler) updateStatusReady(ctx context.Context, sandbox *runtimev1alpha1.ToolSandbox) error {
	sandbox.Status.Url = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		sandbox.Name, sandbox.Namespace, sandbox.Spec.Port)

	meta.SetStatusCondition(&sandbox.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "ToolSandbox is ready",
		ObservedGeneration: sandbox.Generation,
	})

	if err := r.Status().Update(ctx, sandbox); err != nil {
		return fmt.Errorf("failed to update toolsandbox status: %w", err)
	}

	return nil
}

// updateStatusNotReady sets the ToolSandbox status to not Ready
func (r *ToolSandboxReconciler) updateStatusNotReady(ctx context.Context, sandbox *runtimev1alpha1.ToolSandbox, reason, message string) error {
	meta.SetStatusCondition(&sandbox.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: sandbox.Generation,
	})

	if err := r.Status().Update(ctx, sandbox); err != nil {
		return fmt.Errorf("failed to update toolsandbox status: %w", err)
	}

	return nil
}
