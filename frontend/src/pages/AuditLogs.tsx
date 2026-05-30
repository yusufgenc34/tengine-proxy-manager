import { useEffect, useState, useCallback } from 'react'
import { Eye, Settings } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../api/client'
import TableToolbar from '../components/TableToolbar'
import Pagination from '../components/Pagination'
import Offcanvas from '../components/Offcanvas'
import ConfirmDialog from '../components/ConfirmDialog'

interface AuditLog {
  id: number
  user_id: number | null
  action: string
  detail: string
  ip: string
  created_at: string
}

export default function AuditLogs() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [filtersReady, setFiltersReady] = useState(false)
  const [search, setSearch] = useState('')
  const [actionFilter, setActionFilter] = useState('')
  const [userFilter, setUserFilter] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<AuditLog | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [clearAllTarget, setClearAllTarget] = useState(false)

  // Filter options
  const [actionTypes, setActionTypes] = useState<string[]>([])
  const [users, setUsers] = useState<{ id: number; email: string }[]>([])

  // Settings
  const [retentionEnabled, setRetentionEnabled] = useState(() => localStorage.getItem('audit_retention_enabled') !== 'false')
  const [retentionDays, setRetentionDays] = useState(() => {
    const d = localStorage.getItem('audit_retention_days')
    return d ? parseInt(d) : 30
  })

  const saveSettings = (enabled: boolean, days: number) => {
    localStorage.setItem('audit_retention_enabled', String(enabled))
    localStorage.setItem('audit_retention_days', String(days))
    setRetentionEnabled(enabled)
    setRetentionDays(days)
  }

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { page: String(page), limit: '20' }
      if (search) params.action = search
      if (actionFilter) params.action = actionFilter
      if (userFilter) params.user_id = userFilter
      if (dateFrom) params.from = dateFrom
      const { data } = await api.get('/audit-logs', { params })
      setLogs(data.data)
      setTotal(data.total)
    } finally {
      setLoading(false)
    }
  }, [page, search, actionFilter, userFilter, dateFrom])

  const fetchFilterOptions = async () => {
    try {
      const [atRes, uRes] = await Promise.all([
        api.get('/audit-logs/action-types'),
        api.get('/users?limit=100'),
      ])
      setActionTypes(atRes.data.types || [])
      setUsers(uRes.data.data?.map((u: any) => ({ id: u.id, email: u.email })) || [])
    } catch { /* ignore */ }
    setFiltersReady(true)
  }

  useEffect(() => { fetchLogs() }, [fetchLogs])
  useEffect(() => { setPage(1) }, [search, actionFilter, userFilter, dateFrom])
  useEffect(() => { fetchFilterOptions() }, [])

  const handleClearAll = async () => {
    try {
      const { data } = await api.delete('/audit-logs?all=true')
      toast.success(data.message)
      setClearAllTarget(false)
      fetchLogs()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Clear failed')
    }
  }

  const actionLabel = (a: string) => a.replace(/_/g, '.').replace(/\./g, ' ')
  const maskIP = (ip: string) => ip ? ip.replace(/(\d+)$/, '***') : '-'

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Audit Logs</h2>
        <button onClick={() => setSettingsOpen(true)}
          className="p-2 hover:bg-gray-100 rounded-lg text-gray-500 hover:text-gray-700 transition-colors" title="Log Settings">
          <Settings size={20} />
        </button>
      </div>

      <TableToolbar search={search} onSearch={setSearch} searchPlaceholder="Search by action...">
        {actionTypes.length > 0 && (
          <select value={actionFilter} onChange={(e) => setActionFilter(e.target.value)}
            className="px-3 py-2 border rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 max-w-[160px]">
            <option value="">All Actions</option>
            {actionTypes.map(a => (
              <option key={a} value={a}>{actionLabel(a)}</option>
            ))}
          </select>
        )}
        {users.length > 0 && (
          <select value={userFilter} onChange={(e) => setUserFilter(e.target.value)}
            className="px-3 py-2 border rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 max-w-[160px] truncate">
            <option value="">All Users</option>
            {users.map(u => (
              <option key={u.id} value={String(u.id)}>{u.email}</option>
            ))}
          </select>
        )}
        <input type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)}
          className="px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 w-36" title="From date" />
        {(actionFilter || userFilter || dateFrom) && (
          <button onClick={() => { setActionFilter(''); setUserFilter(''); setDateFrom('') }}
            className="text-xs text-gray-400 hover:text-gray-600 whitespace-nowrap">Clear all</button>
        )}
      </TableToolbar>

      {loading || !filtersReady ? (
        <div className="bg-white rounded-xl shadow p-8 text-center text-gray-500">Loading...</div>
      ) : logs.length === 0 ? (
        <div className="bg-white rounded-xl shadow p-8 text-center">
          <p className="text-gray-500 mb-2">No logs found.</p>
          <p className="text-sm text-gray-400">Audit logs track all actions performed in the system.</p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow overflow-x-auto">
          <table className="w-full text-sm min-w-[600px]">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="px-4 py-3 font-medium">Date</th>
                <th className="px-4 py-3 font-medium">Action</th>
                <th className="px-4 py-3 font-medium">User</th>
                <th className="px-4 py-3 font-medium">IP</th>
                <th className="px-4 py-3 font-medium hidden sm:table-cell">Detail</th>
                <th className="px-4 py-3 font-medium w-8"></th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {logs.map((log) => (
                <tr key={log.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-gray-600 whitespace-nowrap text-xs">
                    {new Date(log.created_at).toLocaleString('en-US', { month:'short', day:'numeric', hour:'2-digit', minute:'2-digit' })}
                  </td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-1 bg-gray-100 rounded text-xs font-mono truncate max-w-[120px] block" title={log.action}>
                      {actionLabel(log.action)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-500 text-xs">{log.user_id ?? '-'}</td>
                  <td className="px-4 py-3 text-xs">
                    <span className="blur-[3px] hover:blur-none transition-all duration-150 cursor-default font-mono text-gray-400 select-none">
                      {maskIP(log.ip)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-600 hidden sm:table-cell">
                    <span className="max-w-[200px] truncate block text-xs">{log.detail || '-'}</span>
                  </td>
                  <td className="px-4 py-3">
                    <button onClick={() => setSelected(log)}
                      className="p-1.5 hover:bg-gray-100 rounded text-gray-400 hover:text-blue-600" title="View details">
                      <Eye size={14} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pagination page={page} limit={20} total={total} onPageChange={setPage} />
        </div>
      )}

      {settingsOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
            <h3 className="text-lg font-semibold mb-4">Audit Log Settings</h3>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">Auto-cleanup</p>
                  <p className="text-xs text-gray-400">Delete old logs on server restart</p>
                </div>
                <button onClick={() => saveSettings(!retentionEnabled, retentionDays)}
                  className={`w-11 h-6 rounded-full transition-colors relative ${retentionEnabled ? 'bg-blue-600' : 'bg-gray-300'}`}>
                  <span className={`absolute top-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${retentionEnabled ? 'left-[22px]' : 'left-0.5'}`} />
                </button>
              </div>
              {retentionEnabled && (
                <div>
                  <label className="block text-sm font-medium mb-1">Retention (days)</label>
                  <input type="number" min={1} max={365} value={retentionDays}
                    onChange={(e) => saveSettings(true, Math.max(1, parseInt(e.target.value) || 30))}
                    className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
                </div>
              )}
              <div className="border-t pt-4">
                <button onClick={() => { setSettingsOpen(false); setClearAllTarget(true) }}
                  className="w-full px-4 py-2 text-sm border border-red-200 text-red-600 rounded-lg hover:bg-red-50">
                  Clear all logs
                </button>
              </div>
            </div>
            <div className="flex justify-end mt-4 border-t pt-4">
              <button onClick={() => setSettingsOpen(false)} className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50">Close</button>
            </div>
          </div>
        </div>
      )}

      {clearAllTarget && (
        <ConfirmDialog title="Clear All Audit Logs"
          message="This will permanently delete ALL audit logs. This action cannot be undone."
          onConfirm={handleClearAll} onCancel={() => setClearAllTarget(false)} />
      )}

      {selected && (
        <Offcanvas title="Log Detail" onClose={() => setSelected(null)}>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-1">Date</p>
                <p className="text-sm font-medium">{new Date(selected.created_at).toLocaleString('en-US')}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-1">User ID</p>
                <p className="text-sm font-medium font-mono">{selected.user_id ?? 'System'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-1">IP Address</p>
                <p className="text-sm font-medium font-mono">{selected.ip || '-'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-1">Action</p>
                <span className="px-2 py-0.5 bg-gray-200 rounded text-xs font-mono">{actionLabel(selected.action)}</span>
              </div>
            </div>
            <div className="bg-gray-50 rounded-lg p-3">
              <p className="text-xs text-gray-400 mb-2">Detail</p>
              <pre className="text-sm whitespace-pre-wrap break-all font-sans">{selected.detail || '-'}</pre>
            </div>
          </div>
        </Offcanvas>
      )}
    </div>
  )
}
