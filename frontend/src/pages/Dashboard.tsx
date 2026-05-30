import { useEffect, useState } from 'react'
import { Globe, Lock, Users, Activity, AlertTriangle, Server, Cpu, Box } from 'lucide-react'
import api from '../api/client'

interface Stats {
  proxy_hosts_total: number
  proxy_hosts_enabled: number
  certificates_total: number
  certificates_expiring: number
  users_total: number
  recent_logs: Array<{ id: number; action: string; detail: string; created_at: string }>
  go_version: string
  tengine_version: string
}

export default function Dashboard() {
  const [stats, setStats] = useState<Stats | null>(null)

  const fetchStats = () => {
    api.get('/stats').then(({ data }) => setStats(data))
  }

  useEffect(() => {
    fetchStats()
    const interval = setInterval(fetchStats, 30000)
    return () => clearInterval(interval)
  }, [])

  if (!stats) {
    return <div className="text-gray-500">Loading...</div>
  }

  const cards = [
    { label: 'Proxy Hosts', value: stats.proxy_hosts_total, icon: Globe, color: 'bg-blue-500' },
    { label: 'Active', value: stats.proxy_hosts_enabled, icon: Activity, color: 'bg-green-500' },
    { label: 'Certificates', value: stats.certificates_total, icon: Lock, color: 'bg-purple-500' },
    { label: 'Users', value: stats.users_total, icon: Users, color: 'bg-orange-500' },
  ]

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Dashboard</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map(({ label, value, icon: Icon, color }) => (
          <div key={label} className="bg-white rounded-xl shadow p-5 flex items-center gap-4">
            <div className={`${color} p-3 rounded-lg text-white`}>
              <Icon size={24} />
            </div>
            <div>
              <p className="text-sm text-gray-500">{label}</p>
              <p className="text-2xl font-bold">{value}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Versions */}
      <div className="mt-6">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">System</h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-white rounded-xl shadow p-4 flex items-center gap-3">
            <div className="bg-orange-100 p-2.5 rounded-lg">
              <Server size={20} className="text-orange-600" />
            </div>
            <div className="min-w-0">
              <p className="text-xs text-gray-400">Tengine</p>
              <p className="text-sm font-mono font-medium truncate">{stats.tengine_version?.replace('Tengine version: ', '') || 'unknown'}</p>
            </div>
          </div>
          <div className="bg-white rounded-xl shadow p-4 flex items-center gap-3">
            <div className="bg-cyan-100 p-2.5 rounded-lg">
              <Cpu size={20} className="text-cyan-600" />
            </div>
            <div className="min-w-0">
              <p className="text-xs text-gray-400">Go</p>
              <p className="text-sm font-mono font-medium">{stats.go_version || 'unknown'}</p>
            </div>
          </div>
          <div className="bg-white rounded-xl shadow p-4 flex items-center gap-3">
            <div className="bg-sky-100 p-2.5 rounded-lg">
              <Box size={20} className="text-sky-600" />
            </div>
            <div className="min-w-0">
              <p className="text-xs text-gray-400">React</p>
              <p className="text-sm font-mono font-medium">v18.3.1</p>
            </div>
          </div>
        </div>
      </div>

      {stats.certificates_expiring > 0 && (
        <div className="mt-6 bg-yellow-50 border border-yellow-200 rounded-xl p-4 flex items-start gap-3">
          <AlertTriangle size={20} className="text-yellow-600 mt-0.5" />
          <div>
            <p className="font-medium text-yellow-800">
              {stats.certificates_expiring} certificate{stats.certificates_expiring > 1 ? 's' : ''} expiring soon
            </p>
            <p className="text-sm text-yellow-600 mt-1">
              Check the Certificates page to renew or replace expiring certificates.
            </p>
          </div>
        </div>
      )}

      {stats.recent_logs && stats.recent_logs.length > 0 && (
        <div className="mt-6">
          <h3 className="text-lg font-semibold mb-3">Recent Activity</h3>
          <div className="bg-white rounded-xl shadow overflow-x-auto">
            <table className="w-full text-sm min-w-[400px]">
              <tbody className="divide-y">
                {stats.recent_logs.map((log) => (
                  <tr key={log.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-gray-500 whitespace-nowrap text-xs">
                      {new Date(log.created_at).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-1 bg-gray-100 rounded text-xs font-mono truncate max-w-[100px] block">{log.action}</span>
                    </td>
                    <td className="px-4 py-3 text-gray-600 truncate max-w-md text-xs">{log.detail}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
