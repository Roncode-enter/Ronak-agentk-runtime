import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import LiveAgents from './pages/LiveAgents'
import NewAgent from './pages/NewAgent'
import CostGraph from './pages/CostGraph'
import Attestation from './pages/Attestation'
import Simulation from './pages/Simulation'
import Sidebar from './components/Sidebar'

function isLoggedIn() {
  return !!localStorage.getItem('agentk_token')
}

function ProtectedLayout({ children }: { children: React.ReactNode }) {
  if (!isLoggedIn()) return <Navigate to="/login" />
  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <main className="flex-1 overflow-y-auto p-8">{children}</main>
    </div>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={<ProtectedLayout><Dashboard /></ProtectedLayout>} />
        <Route path="/agents" element={<ProtectedLayout><LiveAgents /></ProtectedLayout>} />
        <Route path="/agents/new" element={<ProtectedLayout><NewAgent /></ProtectedLayout>} />
        <Route path="/cost" element={<ProtectedLayout><CostGraph /></ProtectedLayout>} />
        <Route path="/attestation" element={<ProtectedLayout><Attestation /></ProtectedLayout>} />
        <Route path="/simulation" element={<ProtectedLayout><Simulation /></ProtectedLayout>} />
      </Routes>
    </BrowserRouter>
  )
}
