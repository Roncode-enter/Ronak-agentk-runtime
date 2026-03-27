import { useEffect, useState } from 'react'
import { api } from '../api'
import { Link } from 'react-router-dom'

export default function Dashboard() {
  const [agents, setAgents] = useState<any[]>([])
  const [costs, setCosts] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([api.listAgents(), api.getAllCosts()])
      .then(([a, c]) => { setAgents(a.agents || []); setCosts(c.costs || []) })
      .catch((e) => setError(e.message || 'Failed to load data'))
      .finally(() => setLoading(false))
  }, [])

  const totalCost = costs.reduce((s, c) => s + parseFloat(c.predictedMonthlyCostUSD || '0'), 0)
  const compliantCount = agents.filter(a => a.governanceStatus === 'compliant').length
  const withProofs = agents.filter(a => a.zkProofRoot).length

  if (loading) return (
    <div className="flex items-center justify-center h-full">
      <div className="flex flex-col items-center gap-3">
        <svg className="animate-spin h-8 w-8 text-indigo-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
        <span className="text-slate-400 text-sm">Loading dashboard...</span>
      </div>
    </div>
  )

  return (
    <div className="max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Dashboard</h2>
          <p className="text-slate-500 text-sm mt-1">Overview of your agent runtime environment</p>
        </div>
        <Link to="/agents/new" className="flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-indigo-600 to-purple-600 text-white text-sm font-semibold rounded-xl hover:shadow-lg hover:shadow-indigo-500/25 transition-all duration-200">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
          Deploy Agent
        </Link>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="mb-6 flex items-center gap-3 p-4 bg-red-50 border border-red-200 rounded-xl text-red-700 text-sm">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
          {error}
        </div>
      )}

      {/* Stat Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5 mb-8">
        <StatCard
          title="Total Agents" value={agents.length}
          icon={<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v6m0 6v6m11-7h-6m-6 0H1"/></svg>}
          gradient="from-indigo-500 to-blue-600" lightBg="bg-indigo-50" textColor="text-indigo-700"
        />
        <StatCard
          title="Monthly Cost (est)" value={`$${totalCost.toFixed(2)}`}
          icon={<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>}
          gradient="from-emerald-500 to-teal-600" lightBg="bg-emerald-50" textColor="text-emerald-700"
        />
        <StatCard
          title="Governance Compliant" value={compliantCount}
          icon={<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>}
          gradient="from-blue-500 to-cyan-600" lightBg="bg-blue-50" textColor="text-blue-700"
        />
        <StatCard
          title="ZK-Proof Protected" value={withProofs}
          icon={<svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>}
          gradient="from-purple-500 to-violet-600" lightBg="bg-purple-50" textColor="text-purple-700"
        />
      </div>

      {/* Agent Table */}
      <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
        <div className="flex justify-between items-center px-6 py-5 border-b border-gray-100">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Deployed Agents</h3>
            <p className="text-xs text-slate-400 mt-0.5">{agents.length} agents running across your cluster</p>
          </div>
          <Link to="/agents" className="text-sm text-indigo-600 hover:text-indigo-800 font-medium transition-colors">
            View all &rarr;
          </Link>
        </div>

        {agents.length === 0 ? (
          <div className="text-center py-16 px-6">
            <div className="w-16 h-16 rounded-2xl bg-slate-100 flex items-center justify-center mx-auto mb-4">
              <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#94a3b8" strokeWidth="1.5"><circle cx="12" cy="12" r="3"/><path d="M12 1v6m0 6v6m11-7h-6m-6 0H1"/></svg>
            </div>
            <h4 className="text-gray-700 font-semibold mb-1">No agents deployed yet</h4>
            <p className="text-slate-400 text-sm mb-5">Deploy your first AI agent to get started.</p>
            <Link to="/agents/new" className="inline-flex items-center gap-2 px-5 py-2.5 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition-colors">
              Deploy First Agent
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-[11px] text-slate-400 uppercase tracking-wider bg-slate-50/50">
                  <th className="px-6 py-3 font-semibold">Agent</th>
                  <th className="px-6 py-3 font-semibold">Governance</th>
                  <th className="px-6 py-3 font-semibold">Lifecycle</th>
                  <th className="px-6 py-3 font-semibold">Monthly Cost</th>
                  <th className="px-6 py-3 font-semibold">Proof Mode</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {agents.slice(0, 8).map(a => (
                  <tr key={a.name} className="hover:bg-slate-50/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-indigo-100 to-purple-100 flex items-center justify-center">
                          <span className="text-indigo-600 font-bold text-sm">{a.name?.[0]?.toUpperCase() || 'A'}</span>
                        </div>
                        <span className="font-semibold text-gray-900 text-sm">{a.name}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <Badge text={a.governanceStatus || 'n/a'} variant={a.governanceStatus === 'compliant' ? 'success' : 'warning'} />
                    </td>
                    <td className="px-6 py-4">
                      <Badge text={a.lifecyclePhase || 'n/a'} variant={a.lifecyclePhase === 'stable' ? 'success' : 'info'} />
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-gray-700">${a.predictedMonthlyCostUSD || '0.00'}</td>
                    <td className="px-6 py-4">
                      <span className="text-xs font-mono bg-slate-100 text-slate-600 px-2 py-1 rounded">{a.proofMode || 'merkle-only'}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function StatCard({ title, value, icon, gradient, lightBg, textColor }: {
  title: string; value: any; icon: JSX.Element; gradient: string; lightBg: string; textColor: string
}) {
  return (
    <div className={`rounded-2xl border border-gray-100 bg-white p-5 relative overflow-hidden shadow-sm hover:shadow-md transition-shadow`}>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">{title}</p>
          <p className={`text-3xl font-bold mt-2 ${textColor}`}>{value}</p>
        </div>
        <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${gradient} flex items-center justify-center text-white shadow-lg`}>
          {icon}
        </div>
      </div>
      <div className={`absolute -bottom-4 -right-4 w-24 h-24 rounded-full ${lightBg} opacity-50`} />
    </div>
  )
}

function Badge({ text, variant }: { text: string; variant: 'success' | 'warning' | 'info' | 'danger' }) {
  const styles = {
    success: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    warning: 'bg-amber-50 text-amber-700 border-amber-200',
    info: 'bg-blue-50 text-blue-700 border-blue-200',
    danger: 'bg-red-50 text-red-700 border-red-200',
  }
  return <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold border ${styles[variant]}`}>{text}</span>
}
