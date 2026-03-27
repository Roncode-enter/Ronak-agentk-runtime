import { Link, useLocation } from 'react-router-dom'

const nav = [
  { path: '/', label: 'Dashboard', icon: (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
  )},
  { path: '/agents', label: 'Live Agents', icon: (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 1v6m0 6v6m11-7h-6m-6 0H1m18.07-5.07l-4.24 4.24M9.17 14.83l-4.24 4.24m0-14.14l4.24 4.24m5.66 5.66l4.24 4.24"/></svg>
  )},
  { path: '/agents/new', label: 'New Agent', icon: (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
  )},
  { path: '/cost', label: 'Cost Intelligence', icon: (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
  )},
  { path: '/attestation', label: 'TEE Attestation', icon: (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
  )},
  { path: '/simulation', label: 'Simulation', icon: (
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
  )},
]

export default function Sidebar() {
  const location = useLocation()

  return (
    <aside className="w-72 bg-gradient-to-b from-slate-900 via-slate-900 to-indigo-950 text-white flex flex-col shadow-2xl">
      {/* Logo */}
      <div className="px-6 py-7 border-b border-white/10">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/30">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight">AgentK</h1>
            <p className="text-[11px] text-indigo-300/80 font-medium tracking-wide uppercase">Sovereign Runtime</p>
          </div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-3 px-3 space-y-1">
        <p className="text-[10px] uppercase font-semibold text-slate-500 tracking-wider px-3 pt-3 pb-2">Main Menu</p>
        {nav.slice(0, 3).map((item) => (
          <NavItem key={item.path} item={item} active={location.pathname === item.path} />
        ))}
        <p className="text-[10px] uppercase font-semibold text-slate-500 tracking-wider px-3 pt-5 pb-2">Intelligence</p>
        {nav.slice(3).map((item) => (
          <NavItem key={item.path} item={item} active={location.pathname === item.path} />
        ))}
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-white/10">
        <div className="flex items-center gap-3 px-3 py-2 mb-3 bg-white/5 rounded-lg">
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-emerald-400 to-teal-500 flex items-center justify-center text-xs font-bold">A</div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">Admin</p>
            <p className="text-[11px] text-slate-400 truncate">Managed Tier</p>
          </div>
        </div>
        <button
          onClick={() => { localStorage.removeItem('agentk_token'); window.location.href = '/login' }}
          className="w-full flex items-center justify-center gap-2 text-sm text-slate-400 hover:text-white hover:bg-white/5 py-2 rounded-lg transition-all duration-200"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
          Sign Out
        </button>
      </div>
    </aside>
  )
}

function NavItem({ item, active }: { item: { path: string; label: string; icon: JSX.Element }; active: boolean }) {
  return (
    <Link
      to={item.path}
      className={`flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all duration-200 ${
        active
          ? 'bg-gradient-to-r from-indigo-600/80 to-purple-600/60 text-white font-semibold shadow-lg shadow-indigo-500/20'
          : 'text-slate-400 hover:text-white hover:bg-white/5'
      }`}
    >
      <span className={`flex items-center justify-center w-9 h-9 rounded-lg transition-colors ${
        active ? 'bg-white/20' : 'bg-white/5'
      }`}>
        {item.icon}
      </span>
      {item.label}
    </Link>
  )
}
