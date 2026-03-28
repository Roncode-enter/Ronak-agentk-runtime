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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	runtimev1alpha1 "github.com/agentic-layer/agent-runtime-operator/api/v1alpha1"
	"github.com/agentic-layer/agent-runtime-operator/internal/observability"
	"github.com/agentic-layer/agent-runtime-operator/internal/tee"
	"github.com/agentic-layer/agent-runtime-operator/internal/verifiable"
)

// ConfidentialAgentReconciler reconciles a ConfidentialAgent object.
// It deploys an existing Agent inside a hardware-rooted Trusted Execution Environment (TEE)
// with remote attestation, memory encryption, and attestation sidecars.
type ConfidentialAgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=confidentialagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=confidentialagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runtime.agentic-layer.ai,resources=confidentialagents/finalizers,verbs=update

func (r *ConfidentialAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the ConfidentialAgent instance
	var ca runtimev1alpha1.ConfidentialAgent
	if err := r.Get(ctx, req.NamespacedName, &ca); err != nil {
		if errors.IsNotFound(err) {
			log.Info("ConfidentialAgent resource not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.V(1).Info("Reconciling ConfidentialAgent")

	// Fetch the referenced Agent
	agentNamespace := ca.Spec.AgentRef.Namespace
	if agentNamespace == "" {
		agentNamespace = ca.Namespace
	}
	var agent runtimev1alpha1.Agent
	agentKey := client.ObjectKey{Name: ca.Spec.AgentRef.Name, Namespace: agentNamespace}
	if err := r.Get(ctx, agentKey, &agent); err != nil {
		log.Error(err, "Failed to get referenced Agent", "agent", agentKey)
		if statusErr := r.updateConfidentialAgentStatusNotReady(ctx, &ca, "AgentNotFound", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Ensure TEE-enabled Deployment
	if err := r.ensureConfidentialDeployment(ctx, &ca, &agent); err != nil {
		log.Error(err, "Failed to ensure confidential deployment")
		if statusErr := r.updateConfidentialAgentStatusNotReady(ctx, &ca, "DeploymentFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Ensure Service (same ports as the referenced Agent)
	if err := r.ensureConfidentialService(ctx, &ca, &agent); err != nil {
		log.Error(err, "Failed to ensure confidential service")
		return ctrl.Result{}, err
	}

	// Perform REAL cryptographic attestation verification
	now := time.Now()
	attestationEndpoint := ca.Spec.AttestationEndpoint
	if attestationEndpoint == "" {
		// Default to in-cluster attestation service
		attestationEndpoint = "http://attestation-service.agent-runtime-operator-system.svc.cluster.local:9090"
	}

	// Load the trusted attestation public key from the configured Secret (root of trust).
	// If no Secret is configured, falls back to accepting the key from the attestation
	// response itself (self-signed — NOT recommended for production).
	var trustedPubKeyPEM string
	if ca.Spec.AttestationPublicKeySecretRef != nil {
		secretNamespace := ca.Spec.AttestationPublicKeySecretRef.Namespace
		if secretNamespace == "" {
			secretNamespace = ca.Namespace
		}
		var pubKeySecret corev1.Secret
		secretKey := client.ObjectKey{
			Name:      ca.Spec.AttestationPublicKeySecretRef.Name,
			Namespace: secretNamespace,
		}
		if err := r.Get(ctx, secretKey, &pubKeySecret); err != nil {
			log.Error(err, "Failed to load attestation public key Secret", "secret", secretKey)
			if statusErr := r.updateConfidentialAgentStatusNotReady(ctx, &ca, "PublicKeySecretError",
				fmt.Sprintf("failed to load attestation public key from Secret %s/%s: %v", secretKey.Namespace, secretKey.Name, err)); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		pubKeyBytes, ok := pubKeySecret.Data["publicKey"]
		if !ok || len(pubKeyBytes) == 0 {
			log.Info("Attestation public key Secret missing 'publicKey' field", "secret", secretKey)
			if statusErr := r.updateConfidentialAgentStatusNotReady(ctx, &ca, "PublicKeyMissing",
				fmt.Sprintf("Secret %s/%s does not contain 'publicKey' field", secretKey.Namespace, secretKey.Name)); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		trustedPubKeyPEM = string(pubKeyBytes)
		log.V(1).Info("Using pre-configured attestation root-of-trust key", "secret", secretKey)
	} else {
		log.Info("WARNING: No AttestationPublicKeySecretRef configured — " +
			"attestation will accept the public key from the response (self-signed). " +
			"Configure spec.attestationPublicKeySecretRef for production deployments.")
	}

	attestResult, err := tee.VerifyAttestation(attestationEndpoint, ca.Name, ca.Namespace, trustedPubKeyPEM)
	if err != nil {
		log.Error(err, "Attestation verification failed with error")
		if statusErr := r.updateConfidentialAgentStatusNotReady(ctx, &ca, "AttestationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if !attestResult.Verified {
		log.Info("Attestation verification failed", "reason", attestResult.ErrorMessage)
		// Still update status but with Verified=false; requeue to retry
		ca.Status.AttestationReport = attestResult.ErrorMessage
		ca.Status.TEEProvider = string(ca.Spec.Provider)
		ca.Status.Verified = false
		ca.Status.DeploymentName = fmt.Sprintf("%s-confidential", ca.Name)
		if statusErr := r.updateConfidentialAgentStatusNotReady(ctx, &ca, "AttestationFailed", attestResult.ErrorMessage); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		observability.TEEAgentReadyGauge.WithLabelValues(ca.Name, ca.Namespace, string(ca.Spec.Provider)).Set(0)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Attestation passed — compute the attestation digest from the verified quote
	attestationDigest := attestResult.Digest
	if attestationDigest == "" {
		attestationDigest = verifiable.ComputeAttestationDigest(
			agent.Status.MerkleRoot,
			ca.Name,
			now.Format(time.RFC3339),
		)
	}

	// Update status with real verified attestation
	if err := r.updateConfidentialAgentStatusReady(ctx, &ca, attestationDigest, &now); err != nil {
		return ctrl.Result{}, err
	}

	// Record metrics
	observability.TEEAgentReadyGauge.WithLabelValues(ca.Name, ca.Namespace, string(ca.Spec.Provider)).Set(1)
	observability.TEEAttestationTotal.WithLabelValues(ca.Name, ca.Namespace, string(ca.Spec.Provider)).Inc()

	log.V(1).Info("Reconciled ConfidentialAgent")
	return ctrl.Result{}, nil
}

func (r *ConfidentialAgentReconciler) ensureConfidentialDeployment(
	ctx context.Context,
	ca *runtimev1alpha1.ConfidentialAgent,
	agent *runtimev1alpha1.Agent,
) error {
	log := logf.FromContext(ctx)
	deploymentName := fmt.Sprintf("%s-confidential", ca.Name)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: ca.Namespace,
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
		// Labels
		labels := map[string]string{
			"app":             ca.Name,
			"tee-provider":    string(ca.Spec.Provider),
			"confidential":    "true",
		}
		selectorLabels := map[string]string{
			"app": ca.Name,
		}

		if deployment.CreationTimestamp.IsZero() {
			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			}
		}

		deployment.Labels = labels
		deployment.Spec.Template.Labels = selectorLabels

		// Build TEE pod config
		podConfig := tee.BuildConfidentialPodConfig(&ca.Spec)

		// Set RuntimeClass for Kata Confidential Containers
		deployment.Spec.Template.Spec.RuntimeClassName = podConfig.RuntimeClassName

		// Merge TEE annotations into pod template
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		for k, v := range podConfig.Annotations {
			deployment.Spec.Template.Annotations[k] = v
		}

		// Set replicas from the referenced Agent
		if agent.Spec.Replicas != nil {
			deployment.Spec.Replicas = agent.Spec.Replicas
		}

		// Build the main agent container (copied from Agent spec)
		agentContainer := findContainerByName(&deployment.Spec.Template.Spec, agentContainerName)
		if agentContainer == nil {
			newContainer := corev1.Container{Name: agentContainerName}
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, newContainer)
			agentContainer = &deployment.Spec.Template.Spec.Containers[len(deployment.Spec.Template.Spec.Containers)-1]
		}
		if agent.Spec.Image != "" {
			agentContainer.Image = agent.Spec.Image
		} else {
			agentContainer.Image = defaultTemplateImageAdk
		}
		agentContainer.Env = agent.Spec.Env
		agentContainer.SecurityContext = tee.BuildSecurityContext()

		// Build container ports from agent protocols
		containerPorts := make([]corev1.ContainerPort, 0, len(agent.Spec.Protocols))
		for _, protocol := range agent.Spec.Protocols {
			containerPorts = append(containerPorts, corev1.ContainerPort{
				Name:          protocol.Name,
				ContainerPort: protocol.Port,
				Protocol:      corev1.ProtocolTCP,
			})
		}
		agentContainer.Ports = containerPorts

		// Inject attestation sidecar
		attestationContainer := tee.BuildAttestationSidecar(&ca.Spec)
		existing := findContainerByName(&deployment.Spec.Template.Spec, tee.AttestationSidecarName)
		if existing == nil {
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, attestationContainer)
		} else {
			existing.Image = attestationContainer.Image
			existing.Env = attestationContainer.Env
			existing.Resources = attestationContainer.Resources
		}

		return ctrl.SetControllerReference(ca, deployment, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("Confidential Deployment reconciled", "operation", op)
	}

	return nil
}

func (r *ConfidentialAgentReconciler) ensureConfidentialService(
	ctx context.Context,
	ca *runtimev1alpha1.ConfidentialAgent,
	agent *runtimev1alpha1.Agent,
) error {
	log := logf.FromContext(ctx)

	if len(agent.Spec.Protocols) == 0 {
		return nil
	}

	serviceName := fmt.Sprintf("%s-confidential", ca.Name)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ca.Namespace,
		},
	}

	if op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = map[string]string{
			"app":          ca.Name,
			"confidential": "true",
		}
		service.Spec.Selector = map[string]string{
			"app": ca.Name,
		}
		service.Spec.Type = corev1.ServiceTypeClusterIP

		// Build service ports from agent protocols
		servicePorts := make([]corev1.ServicePort, 0, len(agent.Spec.Protocols))
		for _, protocol := range agent.Spec.Protocols {
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:     protocol.Name,
				Port:     protocol.Port,
				Protocol: corev1.ProtocolTCP,
			})
		}
		service.Spec.Ports = servicePorts

		return ctrl.SetControllerReference(ca, service, r.Scheme)
	}); err != nil {
		return err
	} else if op != controllerutil.OperationResultNone {
		log.Info("Confidential Service reconciled", "operation", op)
	}

	return nil
}

func (r *ConfidentialAgentReconciler) updateConfidentialAgentStatusReady(
	ctx context.Context,
	ca *runtimev1alpha1.ConfidentialAgent,
	attestationDigest string,
	attestationTime *time.Time,
) error {
	ca.Status.AttestationReport = attestationDigest
	ca.Status.TEEProvider = string(ca.Spec.Provider)
	ca.Status.Verified = true
	t := metav1.NewTime(*attestationTime)
	ca.Status.LastAttestationTime = &t
	ca.Status.DeploymentName = fmt.Sprintf("%s-confidential", ca.Name)

	meta.SetStatusCondition(&ca.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "ConfidentialAgent is ready with TEE attestation",
		ObservedGeneration: ca.Generation,
	})

	return r.Status().Update(ctx, ca)
}

func (r *ConfidentialAgentReconciler) updateConfidentialAgentStatusNotReady(
	ctx context.Context,
	ca *runtimev1alpha1.ConfidentialAgent,
	reason string,
	message string,
) error {
	ca.Status.Verified = false

	meta.SetStatusCondition(&ca.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: ca.Generation,
	})

	return r.Status().Update(ctx, ca)
}

// findConfidentialAgentsReferencingAgent returns reconcile requests for all ConfidentialAgents
// that reference the given Agent.
func (r *ConfidentialAgentReconciler) findConfidentialAgentsReferencingAgent(ctx context.Context, obj client.Object) []reconcile.Request {
	var caList runtimev1alpha1.ConfidentialAgentList
	if err := r.List(ctx, &caList); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, ca := range caList.Items {
		agentNs := ca.Spec.AgentRef.Namespace
		if agentNs == "" {
			agentNs = ca.Namespace
		}
		if ca.Spec.AgentRef.Name == obj.GetName() && agentNs == obj.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(&ca),
			})
		}
	}
	return requests
}

func (r *ConfidentialAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.ConfidentialAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Watches(&runtimev1alpha1.Agent{}, handler.EnqueueRequestsFromMapFunc(r.findConfidentialAgentsReferencingAgent)).
		Complete(r)
}
