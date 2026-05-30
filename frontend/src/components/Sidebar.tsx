import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  Globe,
  ShieldCheck,
  Lock,
  Users,
  FileText,
  X,
  LogOut,
  User,
  ChevronUp,
  Fingerprint,
  Settings,
} from 'lucide-react'
import { useAuthStore } from '../store/auth'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/proxy-hosts', icon: Globe, label: 'Proxy Hosts' },
  { to: '/certificates', icon: Lock, label: 'Certificates' },
  { to: '/access-lists', icon: ShieldCheck, label: 'Access Lists' },
  { to: '/users', icon: Users, label: 'Users' },
  { to: '/audit-logs', icon: FileText, label: 'Audit Logs' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

interface Props {
  open: boolean
  onClose: () => void
}

export default function Sidebar({ open, onClose }: Props) {
  const { email, logout } = useAuthStore()
  const navigate = useNavigate()
  const [userOpen, setUserOpen] = useState(false)

  return (
    <>
      {open && <div className="fixed inset-0 bg-black/50 z-40 lg:hidden" onClick={onClose} />}

      <aside className={`fixed lg:sticky top-0 left-0 z-50 w-64 h-screen bg-gray-900 text-white flex flex-col transition-transform duration-200 ${
        open ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
      }`}>
        <div className="p-4 border-b border-gray-700 flex items-center justify-between">
          <h1 className="text-lg font-bold tracking-tight">Tengine Proxy Manager</h1>
          <button onClick={onClose} className="p-1 hover:bg-gray-800 rounded lg:hidden"><X size={18} /></button>
        </div>

        <nav className="flex-1 p-2 overflow-y-auto">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink key={to} to={to} end={to === '/'} onClick={onClose}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg mb-1 transition-colors ${
                  isActive ? 'bg-blue-600 text-white' : 'text-gray-300 hover:bg-gray-800'
                }`}>
              <Icon size={18} /><span className="text-sm">{label}</span>
            </NavLink>
          ))}
        </nav>

        {email && (
          <div className="relative border-t border-gray-700">
            {userOpen && (
              <div className="absolute bottom-full left-0 right-0 mb-1 mx-2 bg-gray-800 rounded-lg border border-gray-700 shadow-lg overflow-hidden">
                <button
                  onClick={() => { navigate('/security'); setUserOpen(false); onClose() }}
                  className="flex items-center gap-2 w-full px-3 py-2.5 text-sm text-gray-300 hover:bg-gray-700 transition-colors"
                >
                  <Fingerprint size={16} /> Security
                </button>
                <button onClick={() => { logout(); setUserOpen(false) }}
                  className="flex items-center gap-2 w-full px-3 py-2.5 text-sm text-gray-300 hover:bg-gray-700 hover:text-red-400 transition-colors">
                  <LogOut size={16} /> Logout
                </button>
              </div>
            )}
            <button onClick={() => setUserOpen(!userOpen)}
              className="flex items-center gap-2 w-full px-4 py-3 text-sm text-gray-400 hover:bg-gray-800 transition-colors">
              <div className="w-7 h-7 bg-gray-700 rounded-full flex items-center justify-center shrink-0"><User size={14} /></div>
              <span className="truncate flex-1 text-left">{email}</span>
              <ChevronUp size={14} className={`shrink-0 transition-transform ${userOpen ? '' : 'rotate-180'}`} />
            </button>
          </div>
        )}
      </aside>
    </>
  )
}
