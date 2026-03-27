import { useState } from 'react'
import { api } from '../api'

export default function Simulation() {
  const [form, setForm] = useState({ name: 'test-agent', replicas: 1, maxMonthlyCost: 50, verifiableEnabled: false })
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  const update = (key: string, value: any) => setForm({ ...form, [key]: value })

  const runPreview = async () => {
    setLoading(true)
    try {
      const data = await api.previewSimulation(form)
      setResult(data)
    } catch (err: any) {
      setResult({ error: err.message })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Simulation Preview</h2>
      <p className="text-gray-500 mb-6">Preview what resources will be created before deploying an agent.</p>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Configuration</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Agent Name</label>
              <input value={form.name} onChange={e => update('name', e.target.value)}
                className="w-full px-4 py-2 rounded-lg border border-gray-300 text-sm" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Replicas</label>
              <input type="number" value={form.replicas} onChange={e => update('replicas', Number(e.target.value))}
                className="w-full px-4 py-2 rounded-lg border border-gray-300 text-sm" min={1} max={10} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Max Monthly Cost ($)</label>
              <input type="number" value={form.maxMonthlyCost} onChange={e => update('maxMonthlyCost', Number(e.target.value))}
                className="w-full px-4 py-2 rounded-lg border border-gray-300 text-sm" />
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={form.verifiableEnabled}
                onChange={e => update('verifiableEnabled', e.target.checked)} className="w-4 h-4 rounded" />
              <span className="text-sm text-gray-700">Enable ZK Proofs</span>
            </label>
          </div>
          <button onClick={runPreview} disabled={loading}
            className="mt-6 w-full py-3 bg-agentk-600 text-white rounded-lg hover:bg-agentk-700 transition disabled:opacity-50">
            {loading ? 'Simulating...' : 'Run Simulation'}
          </button>
        </div>

        {result && (
          <div className="space-y-4">
            {result.error ? (
              <div className="bg-red-50 border border-red-200 rounded-xl p-6">
                <p className="text-red-600">{result.error}</p>
              </div>
            ) : (
              <>
                <div className="bg-white rounded-xl border border-gray-200 p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-3">Estimated Resources</h3>
                  <div className="grid grid-cols-3 gap-4 text-center">
                    <div className="bg-indigo-50 rounded-lg p-4">
                      <p className="text-2xl font-bold text-indigo-600">{result.estimatedResources?.pods}</p>
                      <p className="text-xs text-gray-500">Pods</p>
                    </div>
                    <div className="bg-green-50 rounded-lg p-4">
                      <p className="text-2xl font-bold text-green-600">{result.estimatedResources?.services}</p>
                      <p className="text-xs text-gray-500">Services</p>
                    </div>
                    <div className="bg-purple-50 rounded-lg p-4">
                      <p className="text-2xl font-bold text-purple-600">{result.estimatedResources?.containers_per_pod}</p>
                      <p className="text-xs text-gray-500">Containers/Pod</p>
                    </div>
                  </div>
                  <p className="mt-4 text-center text-lg font-medium text-gray-700">
                    Estimated: <span className="text-green-600">{result.estimatedMonthlyCost}</span>/month
                  </p>
                </div>

                {result.warnings?.length > 0 && (
                  <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-6">
                    <h4 className="font-medium text-yellow-800 mb-2">Warnings</h4>
                    <ul className="list-disc list-inside space-y-1">
                      {result.warnings.map((w: string, i: number) => (
                        <li key={i} className="text-sm text-yellow-700">{w}</li>
                      ))}
                    </ul>
                  </div>
                )}

                <div className="bg-white rounded-xl border border-gray-200 p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-3">Generated YAML</h3>
                  <pre className="bg-gray-50 rounded-lg p-4 text-xs font-mono text-gray-600 overflow-x-auto max-h-96">
                    {result.yaml}
                  </pre>
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
