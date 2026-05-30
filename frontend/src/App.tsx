import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { useEffect, useState, useCallback } from 'react'
import { Menu } from 'lucide-react'
import toast from 'react-hot-toast'
import { useAuthStore } from './store/auth'
import { useIdleTimeout } from './hooks/useIdleTimeout'
import api from './api/client'
import Sidebar from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import ProxyHosts from './pages/ProxyHosts'
import Certificates from './pages/Certificates'
import AccessLists from './pages/AccessLists'
import Users from './pages/Users'
import AuditLogs from './pages/AuditLogs'
import Security from './pages/Security'
import Settings from './pages/Settings'
import DefaultServer from './pages/settings/DefaultServer'
import Login from './pages/Login'
import Setup from './pages/Setup'

function ProtectedLayout() {
  const { isAuthenticated, logout } = useAuthStore()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const handleIdle = useCallback(() => {
    toast.error('Session expired due to inactivity')
    logout()
  }, [logout])

  useIdleTimeout(handleIdle, 15 * 60 * 1000) // 15 min

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
      <div className="flex-1 flex flex-col min-w-0">
        <div className="h-14 border-b border-gray-200 flex items-center gap-3 px-4 lg:hidden shrink-0">
          <button onClick={() => setSidebarOpen(true)} className="p-1.5 hover:bg-gray-100 rounded-lg">
            <Menu size={20} />
          </button>
          <span className="text-lg font-bold">TPM</span>
        </div>
        <main className="flex-1 overflow-y-auto p-4 lg:p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/proxy-hosts" element={<ProxyHosts />} />
            <Route path="/certificates" element={<Certificates />} />
            <Route path="/access-lists" element={<AccessLists />} />
            <Route path="/users" element={<Users />} />
            <Route path="/audit-logs" element={<AuditLogs />} />
            <Route path="/security" element={<Security />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/settings/default-server" element={<DefaultServer />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}

export default function App() {
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null)

  useEffect(() => {
    api.get('/setup/status')
      .then(({ data }) => setSetupRequired(data.setup_required))
      .catch(() => setSetupRequired(false))
  }, [])

  if (setupRequired === null) {
    return <div className="min-h-screen flex items-center justify-center text-gray-500">Loading...</div>
  }

  return (
    <BrowserRouter>
      <Toaster position="top-right" />
      <Routes>
        {setupRequired ? (
          <Route path="*" element={<Setup onComplete={() => setSetupRequired(false)} />} />
        ) : (
          <>
            <Route path="/login" element={<Login />} />
            <Route path="/*" element={<ProtectedLayout />} />
          </>
        )}
      </Routes>
    </BrowserRouter>
  )
}
