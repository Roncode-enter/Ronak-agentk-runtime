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

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/agentic-layer/agent-runtime-operator/test/utils"
)

var _ = Describe("ToolSandbox and Policy", Ordered, func() {
	const (
		sampleFile    = "examples/agent-with-sandbox.yaml"
		testNamespace = "default"

		// Resource names from agent-with-sandbox.yaml
		sandboxName = "secure-sandbox"
		policyName  = "agent-security"
		agentName   = "secure-coding-assistant"
	)

	BeforeAll(func() {
		By("applying the ToolSandbox + Policy + Agent sample")
		_, err := utils.Run(exec.Command("kubectl", "apply", "-f", sampleFile))
		Expect(err).NotTo(HaveOccurred(), "Failed to apply agent-with-sandbox sample")
	})

	AfterAll(func() {
		By("cleaning up the ToolSandbox + Policy + Agent sample")
		_, _ = utils.Run(exec.Command("kubectl", "delete", "-f", sampleFile, "--ignore-not-found=true"))
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			fetchControllerManagerPodLogs()
			fetchKubernetesEvents()
		}
	})

	// ─── Test 1: ToolSandbox CRD gets created and becomes Ready ───

	It("should create ToolSandbox and reach Ready status", func() {
		By("waiting for ToolSandbox to become Ready")
		Eventually(func(g Gomega) {
			output, err := utils.Run(exec.Command("kubectl", "get", "toolsandbox", sandboxName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}"))
			g.Expect(err).NotTo(HaveOccurred(), "Failed to get ToolSandbox status")
			g.Expect(output).To(Equal("True"), "ToolSandbox should be Ready")
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("verifying ToolSandbox has a service URL")
		output, err := utils.Run(exec.Command("kubectl", "get", "toolsandbox", sandboxName,
			"-n", testNamespace,
			"-o", "jsonpath={.status.url}"))
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("svc.cluster.local"), "ToolSandbox should have a cluster-local URL")

		By("verifying ToolSandbox Deployment was created")
		output, err = utils.Run(exec.Command("kubectl", "get", "deployment", sandboxName,
			"-n", testNamespace,
			"-o", "jsonpath={.spec.template.spec.containers[0].name}"))
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(Equal("wasm-sandbox"), "ToolSandbox deployment should have wasm-sandbox container")
	})

	// ─── Test 2: Policy CRD gets created, validated, and reports rule count ───

	It("should create Policy and validate rules", func() {
		By("waiting for Policy to become Ready")
		Eventually(func(g Gomega) {
			output, err := utils.Run(exec.Command("kubectl", "get", "policy", policyName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}"))
			g.Expect(err).NotTo(HaveOccurred(), "Failed to get Policy status")
			g.Expect(output).To(Equal("True"), "Policy should be Ready")
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("verifying Policy reports correct rule count")
		output, err := utils.Run(exec.Command("kubectl", "get", "policy", policyName,
			"-n", testNamespace,
			"-o", "jsonpath={.status.ruleCount}"))
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(Equal("2"), "Policy should report 2 rules")

		By("verifying Policy is enforced")
		output, err = utils.Run(exec.Command("kubectl", "get", "policy", policyName,
			"-n", testNamespace,
			"-o", "jsonpath={.status.enforced}"))
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(Equal("true"), "Policy should be enforced")

		By("verifying Policy print columns show correct type")
		output, err = utils.Run(exec.Command("kubectl", "get", "policy", policyName,
			"-n", testNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("hybrid"), "Policy kubectl output should show type=hybrid")
		Expect(output).To(ContainSubstring("enforcing"), "Policy kubectl output should show enforcement=enforcing")
	})

	// ─── Test 3: Agent pod has WASM sidecar container injected ───

	It("should inject WASM sidecar into Agent pod", func() {
		By("waiting for Agent deployment to be ready")
		Eventually(func(g Gomega) {
			output, err := utils.Run(exec.Command("kubectl", "get", "deployment", agentName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.readyReplicas}"))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("1"), "Agent should have 1 ready replica")
		}, 3*time.Minute, 5*time.Second).Should(Succeed())

		By("checking that pod has the wasm-sandbox sidecar container")
		output, err := utils.Run(exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("app=%s", agentName),
			"-n", testNamespace,
			"-o", "jsonpath={.items[0].spec.containers[*].name}"))
		Expect(err).NotTo(HaveOccurred(), "Failed to get pod containers")

		containerNames := strings.Fields(output)
		Expect(containerNames).To(ContainElement("agent"),
			"Pod should have the main 'agent' container")
		Expect(containerNames).To(ContainElement("wasm-sandbox"),
			"Pod should have the 'wasm-sandbox' sidecar container (WasmEdge)")

		By("checking that the wasm-sandbox container uses the correct image")
		output, err = utils.Run(exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("app=%s", agentName),
			"-n", testNamespace,
			"-o", "jsonpath={.items[0].spec.containers[?(@.name=='wasm-sandbox')].image}"))
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("wasmedge"), "WASM sidecar should use wasmedge image")
	})

	// ─── Test 4: Agent pod has eBPF init container and OPA sidecar ───

	It("should inject eBPF init container and OPA sidecar into Agent pod", func() {
		By("checking that pod has the OPA policy sidecar container")
		output, err := utils.Run(exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("app=%s", agentName),
			"-n", testNamespace,
			"-o", "jsonpath={.items[0].spec.containers[*].name}"))
		Expect(err).NotTo(HaveOccurred())

		containerNames := strings.Fields(output)
		Expect(containerNames).To(ContainElement("opa-policy"),
			"Pod should have the 'opa-policy' sidecar for OPA enforcement")

		By("checking that pod has the eBPF init container")
		output, err = utils.Run(exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("app=%s", agentName),
			"-n", testNamespace,
			"-o", "jsonpath={.items[0].spec.initContainers[*].name}"))
		Expect(err).NotTo(HaveOccurred())

		initContainerNames := strings.Fields(output)
		Expect(initContainerNames).To(ContainElement("ebpf-probe"),
			"Pod should have the 'ebpf-probe' init container for eBPF enforcement")
	})

	// ─── Test 5: Operator logs show reconciliation of new resources ───

	It("should show ToolSandbox and Policy reconciliation in operator logs", func() {
		By("checking operator logs for ToolSandbox reconciliation")
		output, err := utils.Run(exec.Command("kubectl", "logs",
			"-l", "control-plane=controller-manager",
			"-n", namespace,
			"--tail=200"))
		Expect(err).NotTo(HaveOccurred(), "Failed to get operator logs")

		Expect(output).To(ContainSubstring("Reconciling ToolSandbox"),
			"Operator logs should show ToolSandbox reconciliation")
		Expect(output).To(ContainSubstring("Reconciling Policy"),
			"Operator logs should show Policy reconciliation")
	})
})
