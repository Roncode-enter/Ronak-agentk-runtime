import { useEffect, useState } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts'
import { api } from '../api'

export default function CostGraph() {
  const [costs, setCosts] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getAllCosts().then(r => setCosts(r.costs || [])).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const chartData = costs.map(c => ({
    name: c.name,
    cost: parseFloat(c.predictedMonthlyCostUSD || '0'),
    tokens: c.tokensUsed || 0,
    action: c.costAction || 'none',
  }))

  const totalCost = chartData.reduce((s, d) => s + d.cost, 0)
  const totalTokens = chartData.reduce((s, d) => s + d.tokens, 0)

  if (loading) return <div className="text-gray-400 text-center py-20">Loading cost data...</div>

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Cost Intelligence</h2>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <p className="text-sm text-gray-500">Total Monthly Cost</p>
          <p className="text-3xl font-bold text-green-600 mt-1">${totalCost.toFixed(2)}</p>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <p className="text-sm text-gray-500">Total Tokens Used</p>
          <p className="text-3xl font-bold text-indigo-600 mt-1">{totalTokens.toLocaleString()}</p>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <p className="text-sm text-gray-500">Agents Monitored</p>
          <p className="text-3xl font-bold text-gray-900 mt-1">{costs.length}</p>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Cost per Agent</h3>
        {chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={400}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="name" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} label={{ value: '$/month', angle: -90, position: 'insideLeft' }} />
              <Tooltip formatter={(value: number) => [`$${value.toFixed(4)}`, 'Monthly Cost']} />
              <Bar dataKey="cost" radius={[8, 8, 0, 0]}>
                {chartData.map((entry, i) => (
                  <Cell key={i} fill={entry.action === 'none' ? '#6366f1' : entry.action === 'downgraded' ? '#f59e0b' : '#ef4444'} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <p className="text-gray-400 text-center py-10">No cost data available. Deploy agents with costBudget configured.</p>
        )}
      </div>

      <div className="mt-6 bg-white rounded-xl border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Agent Cost Details</h3>
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-xs text-gray-500 uppercase border-b">
              <th className="pb-3 pr-4">Agent</th>
              <th className="pb-3 pr-4">Predicted Cost</th>
              <th className="pb-3 pr-4">Tokens</th>
              <th className="pb-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {costs.map(c => (
              <tr key={c.name} className="border-b border-gray-100">
                <td className="py-3 pr-4 font-medium">{c.name}</td>
                <td className="py-3 pr-4">${c.predictedMonthlyCostUSD || '0.00'}</td>
                <td className="py-3 pr-4">{(c.tokensUsed || 0).toLocaleString()}</td>
                <td className="py-3">
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                    c.costAction === 'none' ? 'bg-green-100 text-green-700' :
                    c.costAction === 'downgraded' ? 'bg-yellow-100 text-yellow-700' :
                    'bg-red-100 text-red-700'
                  }`}>{c.costAction || 'none'}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
