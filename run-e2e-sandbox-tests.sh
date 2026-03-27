#!/bin/bash
# ──────────────────────────────────────────────────────────────
#  ToolSandbox + Policy E2E Tests
#  Applies example YAML, checks WASM sidecar, eBPF init container,
#  OPA sidecar, policy status, and operator logs.
#  Prints "ALL TESTS PASSED" or shows which test failed.
# ──────────────────────────────────────────────────────────────
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

PASS=0
FAIL=0
FAILED_TESTS=""

pass() {
    echo "  ✅ PASSED: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo "  ❌ FAILED: $1"
    echo "     Reason: $2"
    FAIL=$((FAIL + 1))
    FAILED_TESTS="$FAILED_TESTS\n  - $1: $2"
}

echo "============================================"
echo "  ToolSandbox + Policy E2E Tests"
echo "============================================"
echo ""

# ─── Setup: Apply example resources ───
echo "--- Applying example resources ---"
kubectl apply -f examples/agent-with-sandbox.yaml
echo ""
echo "Waiting 15 seconds for reconciliation..."
sleep 15
echo ""

# ─── Test 1: ToolSandbox becomes Ready ───
echo "--- Test 1: ToolSandbox becomes Ready ---"
SANDBOX_READY=$(kubectl get toolsandbox secure-sandbox -n default -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
if [ "$SANDBOX_READY" = "True" ]; then
    pass "ToolSandbox 'secure-sandbox' is Ready"
else
    fail "ToolSandbox 'secure-sandbox' is NOT Ready" "Got status: '$SANDBOX_READY'"
fi

# ─── Test 2: ToolSandbox has a service URL ───
echo "--- Test 2: ToolSandbox has service URL ---"
SANDBOX_URL=$(kubectl get toolsandbox secure-sandbox -n default -o jsonpath='{.status.url}' 2>/dev/null || echo "")
if echo "$SANDBOX_URL" | grep -q "svc.cluster.local"; then
    pass "ToolSandbox has cluster-local URL: $SANDBOX_URL"
else
    fail "ToolSandbox missing service URL" "Got: '$SANDBOX_URL'"
fi

# ─── Test 3: ToolSandbox Deployment has wasm-sandbox container ───
echo "--- Test 3: ToolSandbox Deployment has wasm-sandbox container ---"
SANDBOX_CONTAINER=$(kubectl get deployment secure-sandbox -n default -o jsonpath='{.spec.template.spec.containers[0].name}' 2>/dev/null || echo "")
if [ "$SANDBOX_CONTAINER" = "wasm-sandbox" ]; then
    pass "ToolSandbox Deployment has 'wasm-sandbox' container"
else
    fail "ToolSandbox Deployment missing 'wasm-sandbox' container" "Got: '$SANDBOX_CONTAINER'"
fi

# ─── Test 4: Policy becomes Ready ───
echo "--- Test 4: Policy becomes Ready ---"
POLICY_READY=$(kubectl get policy agent-security -n default -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
if [ "$POLICY_READY" = "True" ]; then
    pass "Policy 'agent-security' is Ready"
else
    fail "Policy 'agent-security' is NOT Ready" "Got status: '$POLICY_READY'"
fi

# ─── Test 5: Policy reports correct rule count ───
echo "--- Test 5: Policy reports 2 rules ---"
RULE_COUNT=$(kubectl get policy agent-security -n default -o jsonpath='{.status.ruleCount}' 2>/dev/null || echo "")
if [ "$RULE_COUNT" = "2" ]; then
    pass "Policy reports 2 rules"
else
    fail "Policy rule count incorrect" "Expected 2, got: '$RULE_COUNT'"
fi

# ─── Test 6: Policy is enforced ───
echo "--- Test 6: Policy is enforced ---"
ENFORCED=$(kubectl get policy agent-security -n default -o jsonpath='{.status.enforced}' 2>/dev/null || echo "")
if [ "$ENFORCED" = "true" ]; then
    pass "Policy is enforced"
else
    fail "Policy is NOT enforced" "Got: '$ENFORCED'"
fi

# ─── Test 7: Policy print columns show type and enforcement ───
echo "--- Test 7: Policy kubectl output shows type and enforcement ---"
POLICY_OUTPUT=$(kubectl get policy agent-security -n default 2>/dev/null || echo "")
if echo "$POLICY_OUTPUT" | grep -q "hybrid" && echo "$POLICY_OUTPUT" | grep -q "enforcing"; then
    pass "Policy shows type=hybrid and enforcement=enforcing"
else
    fail "Policy print columns incorrect" "Output: $POLICY_OUTPUT"
fi

# ─── Test 8: Agent pod has wasm-sandbox sidecar ───
echo "--- Test 8: Agent pod has wasm-sandbox sidecar ---"
AGENT_CONTAINERS=$(kubectl get pods -l app=secure-coding-assistant -n default -o jsonpath='{.items[0].spec.containers[*].name}' 2>/dev/null || echo "")
if echo "$AGENT_CONTAINERS" | grep -q "wasm-sandbox"; then
    pass "Agent pod has 'wasm-sandbox' sidecar container"
else
    fail "Agent pod missing 'wasm-sandbox' sidecar" "Containers found: '$AGENT_CONTAINERS'"
fi

# ─── Test 9: Agent pod has opa-policy sidecar ───
echo "--- Test 9: Agent pod has opa-policy sidecar ---"
if echo "$AGENT_CONTAINERS" | grep -q "opa-policy"; then
    pass "Agent pod has 'opa-policy' sidecar container"
else
    fail "Agent pod missing 'opa-policy' sidecar" "Containers found: '$AGENT_CONTAINERS'"
fi

# ─── Test 10: Agent pod has ebpf-probe init container ───
echo "--- Test 10: Agent pod has ebpf-probe init container ---"
INIT_CONTAINERS=$(kubectl get pods -l app=secure-coding-assistant -n default -o jsonpath='{.items[0].spec.initContainers[*].name}' 2>/dev/null || echo "")
if echo "$INIT_CONTAINERS" | grep -q "ebpf-probe"; then
    pass "Agent pod has 'ebpf-probe' init container"
else
    fail "Agent pod missing 'ebpf-probe' init container" "Init containers found: '$INIT_CONTAINERS'"
fi

# ─── Test 11: wasm-sandbox uses wasmedge image ───
echo "--- Test 11: wasm-sandbox uses wasmedge image ---"
WASM_IMAGE=$(kubectl get pods -l app=secure-coding-assistant -n default -o jsonpath='{.items[0].spec.containers[?(@.name=="wasm-sandbox")].image}' 2>/dev/null || echo "")
if echo "$WASM_IMAGE" | grep -q "wasmedge"; then
    pass "wasm-sandbox uses wasmedge image: $WASM_IMAGE"
else
    fail "wasm-sandbox not using wasmedge image" "Got: '$WASM_IMAGE'"
fi

# ─── Test 12: Operator logs show reconciliation ───
echo "--- Test 12: Operator logs show reconciliation ---"
OP_LOGS=$(kubectl logs -l control-plane=controller-manager -n agent-runtime-operator-system --tail=200 2>/dev/null || echo "")
SANDBOX_LOG=$(echo "$OP_LOGS" | grep -c "Reconciling ToolSandbox" || true)
POLICY_LOG=$(echo "$OP_LOGS" | grep -c "Reconciling Policy" || true)
if [ "$SANDBOX_LOG" -gt 0 ] && [ "$POLICY_LOG" -gt 0 ]; then
    pass "Operator logs show ToolSandbox and Policy reconciliation"
else
    fail "Operator logs missing reconciliation messages" "ToolSandbox mentions: $SANDBOX_LOG, Policy mentions: $POLICY_LOG"
fi

# ─── Results ───
echo ""
echo "============================================"
echo "  Results: $PASS passed, $FAIL failed"
echo "============================================"

if [ "$FAIL" -eq 0 ]; then
    echo ""
    echo "  🎉 ALL TESTS PASSED"
    echo ""
else
    echo ""
    echo "  Failed tests:"
    echo -e "$FAILED_TESTS"
    echo ""
fi

# ─── Cleanup ───
echo "--- Cleaning up ---"
kubectl delete -f examples/agent-with-sandbox.yaml --ignore-not-found=true 2>/dev/null || true
echo "Done."
