import { Link, useLocation } from 'react-router-dom'

const nav = [
  { path: '/', label: 'Dashboard', icon: 'H' },
  { path: '/agents', label: 'Live Agents', icon: 'A' },
  { path: '/agents/new', label: 'New Agent', icon: '+' },
  { path: '/cost', label: 'Cost Intelligence', icon: '$' },
  { path: '/attestation', label: 'TEE Attestation', icon: 'S' },
  { path: '/simulation', label: 'Simulation', icon: 'P' },
]

export default function Sidebar() {
  const location = useLocation()

  return (
    <aside className="w-64 bg-agentk-900 text-white flex flex-col">
      <div className="p-6 border-b border-agentk-700">
        <h1 className="text-xl font-bold tracking-tight">AgentK</h1>
        <p className="text-xs text-indigo-300 mt-1">Sovereign Agent Runtime</p>
      </div>
      <nav className="flex-1 py-4">
        {nav.map((item) => (
          <Link
            key={item.path}
            to={item.path}
            className={`flex items-center px-6 py-3 text-sm transition-colors ${
              location.pathname === item.path
                ? 'bg-agentk-700 text-white font-medium'
                : 'text-indigo-200 hover:bg-agentk-700/50 hover:text-white'
            }`}
          >
            <span className="w-8 h-8 rounded-lg bg-agentk-600 flex items-center justify-center text-xs font-bold mr-3">
              {item.icon}
            </span>
            {item.label}
          </Link>
        ))}
      </nav>
      <div className="p-4 border-t border-agentk-700">
        <button
          onClick={() => { localStorage.removeItem('agentk_token'); window.location.href = '/login' }}
          className="w-full text-sm text-indigo-300 hover:text-white py-2 rounded transition-colors"
        >
          Sign Out
        </button>
      </div>
    </aside>
  )
}
