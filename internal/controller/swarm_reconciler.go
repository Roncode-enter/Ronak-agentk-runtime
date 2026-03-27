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
	"github.com/agentic-layer/agent-runtime-operator/internal/swarm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// SwarmReconciler reconciles a Swarm object
type SwarmReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=swarms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=swarms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=swarms/finalizers,verbs=update
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=agents/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SwarmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Swarm instance
	var sw runtimev1alpha1.Swarm
	if err := r.Get(ctx, req.NamespacedName, &sw); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Swarm resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Swarm")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling Swarm", "name", sw.Name, "strategy", sw.Spec.Strategy)

	// Resolve all agent URLs
	agentUrls, readyCount, totalCount := r.resolveSwarmAgents(ctx, &sw)

	// Ensure ConfigMap with coordinator config
	if err := r.ensureConfigMap(ctx, &sw, agentUrls); err != nil {
		log.Error(err, "Failed to ensure ConfigMap")
		if statusErr := r.updateSwarmStatusNotReady(ctx, &sw, "ConfigMapFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status after ConfigMap failure")
		}
		return ctrl.Result{}, err
	}

	// Ensure coordinator Deployment
	if err := r.ensureDeployment(ctx, &sw); err != nil {
		log.Error(err, "Failed to ensure Deployment")
		if statusErr := r.updateSwarmStatusNotReady(ctx, &sw, "DeploymentFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status after Deployment failure")
		}
		return ctrl.Result{}, err
	}

	// Ensure coordinator Service
	if err := r.ensureService(ctx, &sw); err != nil {
		log.Error(err, "Failed to ensure Service")
		if statusErr := r.updateSwarmStatusNotReady(ctx, &sw, "ServiceFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status after Service failure")
		}
		return ctrl.Result{}, err
	}

	// Update status to Ready
	if err := r.updateSwarmStatusReady(ctx, &sw, readyCount, totalCount); err != nil {
		log.Error(err, "Failed to update Swarm status")
		return ctrl.Result{}, err
	}

	log.Info("Reconciled Swarm", "readyAgents", readyCount, "totalAgents", totalCount)

	// Record metrics
	strategy := string(sw.Spec.Strategy)
	observability.SwarmReconcileTotal.WithLabelValues(sw.Name, sw.Namespace, strategy, "success").Inc()
	observability.SwarmReadyAgentsGauge.WithLabelValues(sw.Name, sw.Namespace, strategy).Set(float64(readyCount))
	observability.SwarmTotalAgentsGauge.WithLabelValues(sw.Name, sw.Namespace, strategy).Set(float64(totalCount))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwarmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.Swarm{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&runtimev1alpha1.Agent{},
			handler.EnqueueRequestsFromMapFunc(r.findSwarmsReferencingAgent),
		).
		Named("swarm").
		Complete(r)
}

// resolveSwarmAgents resolves all agent URLs in the swarm.
// Returns a map of agent name → URL, the count of ready agents, and the total count.
func (r *SwarmReconciler) resolveSwarmAgents(ctx context.Context, sw *runtimev1alpha1.Swarm) (map[string]string, int32, int32) {
	log := logf.FromContext(ctx)
	agentUrls := make(map[string]string)
	var readyCount int32
	totalCount := int32(len(sw.Spec.Agents))

	for _, sa := range sw.Spec.Agents {
		url, err := r.resolveSwarmAgentUrl(ctx, sa, sw.Namespace)
		if err != nil {
			log.Info("Agent not ready", "agent", sa.Name, "error", err.Error())
			continue
		}
		agentUrls[sa.Name] = url
		readyCount++
	}

	return agentUrls, readyCount, totalCount
}

// resolveSwarmAgentUrl resolves a single SwarmAgent to its URL.
func (r *SwarmReconciler) resolveSwarmAgentUrl(ctx context.Context, sa runtimev1alpha1.SwarmAgent, parentNamespace string) (string, error) {
	// If URL is provided, use it directly (remote agent)
	if sa.Url != "" {
		return sa.Url, nil
	}

	// Cluster agent reference — resolve by looking up the Agent resource
	if sa.AgentRef == nil {
		return "", fmt.Errorf("agent %q has neither url nor agentRef specified", sa.Name)
	}

	namespace := getNamespaceWithDefault(sa.AgentRef, parentNamespace)

	var agent runtimev1alpha1.Agent
	if err := r.Get(ctx, types.NamespacedName{
		Name:      sa.AgentRef.Name,
		Namespace: namespace,
	}, &agent); err != nil {
		return "", fmt.Errorf("failed to resolve agent %s/%s: %w", namespace, sa.AgentRef.Name, err)
	}

	if agent.Status.Url == "" {
		return "", fmt.Errorf("agent %s/%s has no URL in status (may not be ready)", namespace, sa.AgentRef.Name)
	}

	return agent.Status.Url, nil
}

// ensureConfigMap creates or updates the ConfigMap with the coordinator configuration.
func (r *SwarmReconciler) ensureConfigMap(ctx context.Context, sw *runtimev1alpha1.Swarm, agentUrls map[string]string) error {
	log := logf.FromContext(ctx)

	configMapName := swarm.ConfigMapName(sw.Name)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: sw.Namespace,
		},
	}

	if op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		managedLabels := map[string]string{
			"app":   swarm.CoordinatorName(sw.Name),
			"swarm": sw.Name,
		}

		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}
		applyCommonMetadataToObjectMeta(&cm.ObjectMeta, sw.Spec.CommonMetadata)
		maps.Copy(cm.Labels, managedLabels)

		cm.Data = map[string]string{
			swarm.ConfigFileName: swarm.BuildCoordinatorConfig(sw, agentUrls),
		}

		return ctrl.SetControllerReference(sw, cm, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("ConfigMap reconciled", "operation", op)
	}

	return nil
}

// ensureDeployment creates or updates the coordinator Deployment.
func (r *SwarmReconciler) ensureDeployment(ctx context.Context, sw *runtimev1alpha1.Swarm) error {
	log := logf.FromContext(ctx)

	coordinatorName := swarm.CoordinatorName(sw.Name)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coordinatorName,
			Namespace: sw.Namespace,
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
			"app":   coordinatorName,
			"swarm": sw.Name,
		}

		selectorLabels := map[string]string{
			"app": coordinatorName,
		}

		// Set immutable fields only on creation
		if deployment.CreationTimestamp.IsZero() {
			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			}
		}

		// Build pod template labels
		podTemplateLabels, podTemplateAnnotations := buildPodTemplateMetadata(
			selectorLabels, sw.Spec.CommonMetadata, nil,
		)
		deployment.Spec.Template.Labels = podTemplateLabels
		deployment.Spec.Template.Annotations = podTemplateAnnotations

		// Set replicas
		if deployment.Spec.Replicas == nil {
			deployment.Spec.Replicas = new(int32)
		}
		if sw.Spec.Replicas != nil {
			*deployment.Spec.Replicas = *sw.Spec.Replicas
		} else {
			*deployment.Spec.Replicas = 1
		}

		// Merge labels
		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
		}
		applyCommonMetadataToObjectMeta(&deployment.ObjectMeta, sw.Spec.CommonMetadata)
		maps.Copy(deployment.Labels, managedLabels)

		// Build the coordinator container
		coordinatorContainer := swarm.BuildCoordinatorContainer(sw)

		// Update or create container
		container := findContainerByName(&deployment.Spec.Template.Spec, swarm.CoordinatorContainerName)
		if container == nil {
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, coordinatorContainer)
		} else {
			container.Image = coordinatorContainer.Image
			container.Ports = coordinatorContainer.Ports
			container.Env = coordinatorContainer.Env
			container.VolumeMounts = coordinatorContainer.VolumeMounts
			container.Resources = coordinatorContainer.Resources
			container.ReadinessProbe = coordinatorContainer.ReadinessProbe
		}

		// Set up config volume
		configMapName := swarm.ConfigMapName(sw.Name)
		configVolume := swarm.BuildConfigVolume(configMapName)

		// Replace or add the config volume
		found := false
		for i, v := range deployment.Spec.Template.Spec.Volumes {
			if v.Name == swarm.ConfigVolumeName {
				deployment.Spec.Template.Spec.Volumes[i] = configVolume
				found = true
				break
			}
		}
		if !found {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, configVolume)
		}

		// Set owner reference
		return ctrl.SetControllerReference(sw, deployment, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("Deployment reconciled", "operation", op)
	}

	return nil
}

// ensureService creates or updates the coordinator Service.
func (r *SwarmReconciler) ensureService(ctx context.Context, sw *runtimev1alpha1.Swarm) error {
	log := logf.FromContext(ctx)

	coordinatorName := swarm.CoordinatorName(sw.Name)
	port := swarm.GetCoordinatorPort(sw)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coordinatorName,
			Namespace: sw.Namespace,
		},
	}

	if op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		managedLabels := map[string]string{
			"app":   coordinatorName,
			"swarm": sw.Name,
		}

		selectorLabels := map[string]string{
			"app": coordinatorName,
		}

		servicePorts := []corev1.ServicePort{
			{
				Name:       "http",
				Port:       port,
				TargetPort: intstr.FromInt32(port),
				Protocol:   corev1.ProtocolTCP,
			},
		}

		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		applyCommonMetadataToObjectMeta(&service.ObjectMeta, sw.Spec.CommonMetadata)
		maps.Copy(service.Labels, managedLabels)

		service.Spec.Ports = servicePorts
		service.Spec.Selector = selectorLabels
		service.Spec.Type = corev1.ServiceTypeClusterIP

		return ctrl.SetControllerReference(sw, service, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("Service reconciled", "operation", op)
	}

	return nil
}

// updateSwarmStatusReady sets the Swarm status to Ready.
func (r *SwarmReconciler) updateSwarmStatusReady(ctx context.Context, sw *runtimev1alpha1.Swarm, readyAgents, totalAgents int32) error {
	port := swarm.GetCoordinatorPort(sw)
	coordinatorName := swarm.CoordinatorName(sw.Name)

	sw.Status.ReadyAgents = readyAgents
	sw.Status.TotalAgents = totalAgents
	sw.Status.Strategy = string(sw.Spec.Strategy)
	sw.Status.CoordinatorUrl = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		coordinatorName, sw.Namespace, port)

	meta.SetStatusCondition(&sw.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            fmt.Sprintf("Swarm coordinator is ready (%d/%d agents available)", readyAgents, totalAgents),
		ObservedGeneration: sw.Generation,
	})

	if err := r.Status().Update(ctx, sw); err != nil {
		return fmt.Errorf("failed to update swarm status: %w", err)
	}

	return nil
}

// updateSwarmStatusNotReady sets the Swarm status to not Ready.
func (r *SwarmReconciler) updateSwarmStatusNotReady(ctx context.Context, sw *runtimev1alpha1.Swarm, reason, message string) error {
	meta.SetStatusCondition(&sw.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: sw.Generation,
	})

	if err := r.Status().Update(ctx, sw); err != nil {
		return fmt.Errorf("failed to update swarm status: %w", err)
	}

	return nil
}

// findSwarmsReferencingAgent finds all Swarm resources that reference a given agent.
func (r *SwarmReconciler) findSwarmsReferencingAgent(ctx context.Context, obj client.Object) []ctrl.Request {
	agent, ok := obj.(*runtimev1alpha1.Agent)
	if !ok {
		return nil
	}

	var swarmList runtimev1alpha1.SwarmList
	if err := r.List(ctx, &swarmList); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to list swarms for agent watch")
		return nil
	}

	var requests []ctrl.Request
	for _, sw := range swarmList.Items {
		for _, sa := range sw.Spec.Agents {
			if sa.AgentRef != nil && sa.AgentRef.Name == agent.Name {
				namespace := getNamespaceWithDefault(sa.AgentRef, sw.Namespace)
				if namespace == agent.Namespace {
					requests = append(requests, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      sw.Name,
							Namespace: sw.Namespace,
						},
					})
					logf.FromContext(ctx).Info("Enqueuing swarm due to agent change",
						"swarm", sw.Name, "agent", agent.Name)
					break
				}
			}
		}
	}

	return requests
}
