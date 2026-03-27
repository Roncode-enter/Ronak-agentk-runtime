import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'

export default function NewAgent() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    name: '', description: '', instruction: '', model: 'gemini/gemini-2.5-flash',
    replicas: 1, apiKey: '', maxMonthlyCost: '', optimizationMode: '',
    verifiableEnabled: false, proofMode: 'snark-groth16', autonomyLevel: 3,
    strategy: '', promptVersion: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const update = (key: string, value: any) => setForm({ ...form, [key]: value })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true); setError('')
    try {
      await api.createAgent({
        ...form,
        replicas: Number(form.replicas),
        maxMonthlyCost: form.maxMonthlyCost ? Number(form.maxMonthlyCost) : undefined,
        autonomyLevel: Number(form.autonomyLevel),
      })
      navigate('/agents')
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-3xl">
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Create New Agent</h2>
      <form onSubmit={handleSubmit} className="bg-white rounded-xl border border-gray-200 shadow-sm p-8 space-y-6">
        {/* Basic */}
        <Section title="Basic Configuration">
          <Field label="Agent Name" required>
            <input value={form.name} onChange={e => update('name', e.target.value)} className="input" placeholder="my-agent" required />
          </Field>
          <Field label="Description">
            <input value={form.description} onChange={e => update('description', e.target.value)} className="input" placeholder="A helpful assistant" />
          </Field>
          <Field label="System Instruction">
            <textarea value={form.instruction} onChange={e => update('instruction', e.target.value)} className="input h-24" placeholder="You are a helpful AI assistant..." />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Model">
              <input value={form.model} onChange={e => update('model', e.target.value)} className="input" />
            </Field>
            <Field label="Replicas">
              <input type="number" value={form.replicas} onChange={e => update('replicas', e.target.value)} className="input" min={1} max={10} />
            </Field>
          </div>
          <Field label="API Key">
            <input type="password" value={form.apiKey} onChange={e => update('apiKey', e.target.value)} className="input" placeholder="Your Gemini/OpenAI key" />
          </Field>
        </Section>

        {/* Cost */}
        <Section title="Cost Intelligence">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Max Monthly Cost ($)">
              <input value={form.maxMonthlyCost} onChange={e => update('maxMonthlyCost', e.target.value)} className="input" placeholder="100" />
            </Field>
            <Field label="Optimization Mode">
              <select value={form.optimizationMode} onChange={e => update('optimizationMode', e.target.value)} className="input">
                <option value="">None</option>
                <option value="conservative">Conservative (90%)</option>
                <option value="auto">Auto (80%)</option>
                <option value="aggressive">Aggressive (70%)</option>
              </select>
            </Field>
          </div>
        </Section>

        {/* Verifiable */}
        <Section title="Verifiable Execution (ZK Proofs)">
          <label className="flex items-center gap-3 cursor-pointer">
            <input type="checkbox" checked={form.verifiableEnabled} onChange={e => update('verifiableEnabled', e.target.checked)} className="w-5 h-5 rounded" />
            <span className="text-sm font-medium text-gray-700">Enable cryptographic proof chain</span>
          </label>
          {form.verifiableEnabled && (
            <Field label="Proof Mode">
              <select value={form.proofMode} onChange={e => update('proofMode', e.target.value)} className="input">
                <option value="merkle-only">Merkle Only (Free)</option>
                <option value="snark-groth16">Groth16 zk-SNARK (Standard)</option>
                <option value="plonk-universal">PlonK Universal (Premium $49/mo)</option>
              </select>
            </Field>
          )}
        </Section>

        {/* Governance */}
        <Section title="Governance">
          <Field label="Autonomy Level (1-5)">
            <input type="range" min={1} max={5} value={form.autonomyLevel} onChange={e => update('autonomyLevel', e.target.value)} className="w-full" />
            <div className="flex justify-between text-xs text-gray-400 mt-1">
              <span>1: Human required</span><span>3: Compliant</span><span>5: Autonomous</span>
            </div>
          </Field>
        </Section>

        {/* Lifecycle */}
        <Section title="Lifecycle">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Deployment Strategy">
              <select value={form.strategy} onChange={e => update('strategy', e.target.value)} className="input">
                <option value="">Default</option>
                <option value="rolling">Rolling</option>
                <option value="canary">Canary</option>
                <option value="blue-green">Blue-Green</option>
              </select>
            </Field>
            <Field label="Prompt Version">
              <input value={form.promptVersion} onChange={e => update('promptVersion', e.target.value)} className="input" placeholder="v1.0.0" />
            </Field>
          </div>
        </Section>

        {error && <p className="text-red-500 text-sm">{error}</p>}
        <button type="submit" disabled={loading} className="w-full py-3 bg-agentk-600 hover:bg-agentk-700 text-white font-medium rounded-lg transition disabled:opacity-50">
          {loading ? 'Creating...' : 'Create Agent'}
        </button>
      </form>
      <style>{`.input { @apply w-full px-4 py-2.5 rounded-lg border border-gray-300 focus:ring-2 focus:ring-agentk-500 focus:border-transparent outline-none transition text-sm; }`}</style>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-3 pb-2 border-b border-gray-100">{title}</h3>
      <div className="space-y-4">{children}</div>
    </div>
  )
}

function Field({ label, children, required }: { label: string; children: React.ReactNode; required?: boolean }) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-600 mb-1">{label}{required && <span className="text-red-400"> *</span>}</label>
      {children}
    </div>
  )
}
