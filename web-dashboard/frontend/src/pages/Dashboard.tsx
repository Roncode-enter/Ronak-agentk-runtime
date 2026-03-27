import { useEffect, useState } from 'react'
import { api } from '../api'
import { Link } from 'react-router-dom'

export default function Dashboard() {
  const [agents, setAgents] = useState<any[]>([])
  const [costs, setCosts] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.listAgents(), api.getAllCosts()])
      .then(([a, c]) => { setAgents(a.agents || []); setCosts(c.costs || []) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const totalCost = costs.reduce((s, c) => s + parseFloat(c.predictedMonthlyCostUSD || '0'), 0)
  const compliantCount = agents.filter(a => a.governanceStatus === 'compliant').length
  const withProofs = agents.filter(a => a.zkProofRoot).length

  if (loading) return <div className="flex items-center justify-center h-full text-gray-400">Loading...</div>

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h2>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <StatCard title="Total Agents" value={agents.length} color="indigo" />
        <StatCard title="Monthly Cost" value={`$${totalCost.toFixed(2)}`} color="green" />
        <StatCard title="Governance Compliant" value={compliantCount} color="blue" />
        <StatCard title="ZK-Proof Protected" value={withProofs} color="purple" />
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Recent Agents</h3>
          <Link to="/agents/new" className="px-4 py-2 bg-agentk-600 text-white text-sm rounded-lg hover:bg-agentk-700 transition">
            + New Agent
          </Link>
        </div>
        <table className="w-full">
          <thead>
            <tr className="text-left text-xs text-gray-500 uppercase border-b">
              <th className="pb-3 pr-4">Name</th>
              <th className="pb-3 pr-4">Governance</th>
              <th className="pb-3 pr-4">Lifecycle</th>
              <th className="pb-3 pr-4">Cost</th>
              <th className="pb-3">Proof Mode</th>
            </tr>
          </thead>
          <tbody>
            {agents.slice(0, 8).map(a => (
              <tr key={a.name} className="border-b border-gray-100 hover:bg-gray-50">
                <td className="py-3 pr-4 font-medium text-gray-900">{a.name}</td>
                <td className="py-3 pr-4">
                  <Badge text={a.governanceStatus || 'n/a'} color={a.governanceStatus === 'compliant' ? 'green' : 'yellow'} />
                </td>
                <td className="py-3 pr-4">
                  <Badge text={a.lifecyclePhase || 'n/a'} color={a.lifecyclePhase === 'stable' ? 'green' : 'blue'} />
                </td>
                <td className="py-3 pr-4 text-sm text-gray-600">${a.predictedMonthlyCostUSD || '0.00'}</td>
                <td className="py-3 text-sm text-gray-600">{a.proofMode || 'merkle-only'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function StatCard({ title, value, color }: { title: string; value: any; color: string }) {
  const colors: Record<string, string> = {
    indigo: 'bg-indigo-50 text-indigo-700 border-indigo-200',
    green: 'bg-green-50 text-green-700 border-green-200',
    blue: 'bg-blue-50 text-blue-700 border-blue-200',
    purple: 'bg-purple-50 text-purple-700 border-purple-200',
  }
  return (
    <div className={`rounded-xl border p-6 ${colors[color]}`}>
      <p className="text-sm font-medium opacity-75">{title}</p>
      <p className="text-3xl font-bold mt-1">{value}</p>
    </div>
  )
}

function Badge({ text, color }: { text: string; color: string }) {
  const colors: Record<string, string> = {
    green: 'bg-green-100 text-green-700',
    yellow: 'bg-yellow-100 text-yellow-700',
    blue: 'bg-blue-100 text-blue-700',
    red: 'bg-red-100 text-red-700',
  }
  return <span className={`px-2 py-1 rounded-full text-xs font-medium ${colors[color]}`}>{text}</span>
}
