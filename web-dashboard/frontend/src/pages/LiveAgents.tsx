import { useEffect, useState } from 'react'
import { api } from '../api'

export default function LiveAgents() {
  const [agents, setAgents] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = () => {
    setLoading(true)
    api.listAgents().then(r => setAgents(r.agents || [])).catch(() => {}).finally(() => setLoading(false))
  }

  useEffect(() => { refresh(); const t = setInterval(refresh, 10000); return () => clearInterval(t) }, [])

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Live Agents</h2>
        <button onClick={refresh} className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 text-sm transition">
          Refresh
        </button>
      </div>

      {loading && agents.length === 0 ? (
        <div className="text-gray-400 text-center py-20">Loading agents...</div>
      ) : (
        <div className="grid gap-4">
          {agents.map(a => (
            <div key={a.name} className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 hover:shadow-md transition">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">{a.name}</h3>
                  <p className="text-sm text-gray-500 mt-1">Framework: {a.framework} | Replicas: {a.replicas}</p>
                </div>
                <div className="flex gap-2">
                  {a.governanceStatus && (
                    <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                      a.governanceStatus === 'compliant' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'
                    }`}>{a.governanceStatus}</span>
                  )}
                  {a.lifecyclePhase && (
                    <span className="px-3 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-700">{a.lifecyclePhase}</span>
                  )}
                </div>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4 text-sm">
                <div>
                  <p className="text-gray-400">Monthly Cost</p>
                  <p className="font-medium">${a.predictedMonthlyCostUSD || '0.00'}</p>
                </div>
                <div>
                  <p className="text-gray-400">Tokens Used</p>
                  <p className="font-medium">{(a.tokensUsed || 0).toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-gray-400">Proof Mode</p>
                  <p className="font-medium">{a.proofMode || 'merkle-only'}</p>
                </div>
                <div>
                  <p className="text-gray-400">Merkle Root</p>
                  <p className="font-mono text-xs truncate">{a.merkleRoot || 'n/a'}</p>
                </div>
              </div>
              {a.zkProofRoot && (
                <div className="mt-3 p-3 bg-purple-50 rounded-lg">
                  <p className="text-xs text-purple-600 font-medium">ZK Proof Root</p>
                  <p className="font-mono text-xs text-purple-800 truncate">{a.zkProofRoot}</p>
                </div>
              )}
              <div className="mt-4 flex justify-end">
                <button
                  onClick={() => { if (confirm(`Delete agent ${a.name}?`)) api.deleteAgent(a.name).then(refresh) }}
                  className="px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded transition"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
