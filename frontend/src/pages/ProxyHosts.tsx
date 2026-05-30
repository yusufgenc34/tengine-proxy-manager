import { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, ToggleLeft, ToggleRight, ChevronDown, ChevronUp } from 'lucide-react'
import toast from 'react-hot-toast'
import { useProxyStore, type ProxyHost } from '../store/proxy'
import ProxyHostModal from '../components/ProxyHostModal'
import TableToolbar from '../components/TableToolbar'
import Pagination from '../components/Pagination'
import ConfirmDialog from '../components/ConfirmDialog'

export default function ProxyHosts() {
  const { hosts, total, loading, fetch, remove, toggle } = useProxyStore()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ProxyHost | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ProxyHost | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [sslFilter, setSslFilter] = useState('')
  const [page, setPage] = useState(1)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const loadData = useCallback(() => {
    const params: Record<string, string> = { page: String(page), limit: '20' }
    if (search) params.search = search
    if (statusFilter) params.enabled = statusFilter
    if (sslFilter) params.ssl = sslFilter
    fetch(params)
  }, [fetch, page, search, statusFilter, sslFilter])

  useEffect(() => { loadData() }, [loadData])

  useEffect(() => { setPage(1) }, [search, statusFilter, sslFilter])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove(deleteTarget.id)
      toast.success('Deleted')
      setDeleteTarget(null)
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Delete failed')
    }
  }

  const handleToggle = async (host: ProxyHost) => {
    try {
      await toggle(host.id, !host.enabled)
      toast.success(host.enabled ? 'Disabled' : 'Enabled')
      loadData()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Operation failed')
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Proxy Hosts</h2>
        <button
          onClick={() => { setEditing(null); setModalOpen(true) }}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus size={18} /> Add
        </button>
      </div>

      <TableToolbar
        search={search}
        onSearch={setSearch}
        searchPlaceholder="Search domains..."
        filters={[
          {
            key: 'status', value: statusFilter, label: 'Status',
            options: [{ value: '', label: 'All Status' }, { value: 'true', label: 'Active' }, { value: 'false', label: 'Inactive' }],
          },
          {
            key: 'ssl', value: sslFilter, label: 'SSL',
            options: [{ value: '', label: 'All' }, { value: 'true', label: 'SSL' }, { value: 'false', label: 'HTTP' }],
          },
        ]}
        onFilter={(key, v) => key === 'status' ? setStatusFilter(v) : setSslFilter(v)}
      />

      {loading ? (
        <div className="bg-white rounded-xl shadow p-8 text-center text-gray-500">Loading...</div>
      ) : hosts.length === 0 ? (
        <div className="bg-white rounded-xl shadow p-8 text-center">
          <p className="text-gray-500 mb-2">{search ? 'No results found.' : 'No proxy hosts yet.'}</p>
          <p className="text-sm text-gray-400">
            {search ? 'Try a different search term.' : 'Add a proxy host to route traffic from a domain to your backend services.'}
          </p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow overflow-x-auto">
          <table className="w-full text-sm min-w-[600px]">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="px-4 py-3 font-medium w-8"></th>
                <th className="px-4 py-3 font-medium">Domain</th>
                <th className="px-4 py-3 font-medium hidden sm:table-cell">Forward</th>
                <th className="px-4 py-3 font-medium hidden sm:table-cell">SSL</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium hidden md:table-cell">Created</th>
                <th className="px-4 py-3 font-medium hidden md:table-cell">Updated</th>
                <th className="px-4 py-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {hosts.map((host) => {
                const isExpanded = expandedId === host.id
                const hasDetails = host.certificate || host.access_list || host.health_check || !!host.load_balancing
                return (
                  <>
                    <tr key={host.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        {hasDetails && (
                          <button
                            onClick={() => setExpandedId(isExpanded ? null : host.id)}
                            className="p-0.5 hover:bg-gray-200 rounded"
                          >
                            {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                          </button>
                        )}
                      </td>
                      <td className="px-4 py-3 font-medium whitespace-nowrap">{host.domain}</td>
                      <td className="px-4 py-3 text-gray-600 font-mono text-xs hidden sm:table-cell">
                        {host.forward_scheme}://{host.forward_host}:{host.forward_port}
                      </td>
                      <td className="px-4 py-3 hidden sm:table-cell">
                        <span className={`px-2 py-0.5 rounded text-xs ${host.ssl_enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                          {host.ssl_enabled ? 'SSL' : 'HTTP'}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-0.5 rounded text-xs ${host.enabled ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-600'}`}>
                          {host.enabled ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-500 text-xs hidden md:table-cell whitespace-nowrap">
                        {new Date(host.created_at).toLocaleDateString('en-US', { month:'short', day:'numeric' })}
                      </td>
                      <td className="px-4 py-3 text-gray-500 text-xs hidden md:table-cell whitespace-nowrap">
                        {new Date(host.updated_at).toLocaleDateString('en-US', { month:'short', day:'numeric' })}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button onClick={() => handleToggle(host)} className="p-1 hover:text-blue-600" title={host.enabled ? 'Disable' : 'Enable'}>
                            {host.enabled ? <ToggleRight size={18} className="text-green-600" /> : <ToggleLeft size={18} className="text-gray-400" />}
                          </button>
                          <button onClick={() => { setEditing(host); setModalOpen(true) }} className="p-1 hover:text-blue-600" title="Edit">
                            <Pencil size={16} />
                          </button>
                          <button onClick={() => setDeleteTarget(host)} className="p-1 hover:text-red-600" title="Delete">
                            <Trash2 size={16} />
                          </button>
                        </div>
                      </td>
                    </tr>
                    {isExpanded && (
                      <tr key={`${host.id}-details`} className="bg-gray-50">
                        <td></td>
                        <td colSpan={7} className="px-4 py-3">
                          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                            <div className="bg-white rounded-lg border p-3">
                              <p className="text-xs text-gray-400 mb-1">Certificate</p>
                              <p className="text-sm font-medium">{host.certificate ? host.certificate.domain : 'None'}</p>
                            </div>
                            <div className="bg-white rounded-lg border p-3">
                              <p className="text-xs text-gray-400 mb-1">Access List</p>
                              <p className="text-sm font-medium">{host.access_list ? host.access_list.name : 'None'}</p>
                            </div>
                            <div className="bg-white rounded-lg border p-3">
                              <p className="text-xs text-gray-400 mb-1">Health Check</p>
                              <p className="text-sm font-medium">{host.health_check ? 'Enabled' : 'Disabled'}</p>
                            </div>
                            <div className="bg-white rounded-lg border p-3">
                              <p className="text-xs text-gray-400 mb-1">Load Balancing</p>
                              <p className="text-sm font-medium">
                                {host.load_balancing === 'least_conn' ? 'Least Connections' :
                                 host.load_balancing === 'consistent_hash' ? 'Consistent Hash' : 'Round Robin'}
                              </p>
                            </div>
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                )
              })}
            </tbody>
          </table>
          <Pagination page={page} limit={20} total={total} onPageChange={setPage} />
        </div>
      )}

      {modalOpen && (
        <ProxyHostModal
          host={editing}
          onClose={() => { setModalOpen(false); loadData() }}
        />
      )}

      {deleteTarget && (
        <ConfirmDialog
          title="Delete Proxy Host"
          message={`Are you sure you want to delete "${deleteTarget.domain}"? This will also remove the Tengine configuration.`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
