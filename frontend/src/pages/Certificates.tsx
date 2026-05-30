import { useEffect, useState, useCallback } from 'react'
import { Plus, Trash2, RefreshCw, AlertTriangle, Download } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../api/client'
import CertModal from '../components/CertModal'
import TableToolbar from '../components/TableToolbar'
import Pagination from '../components/Pagination'
import ConfirmDialog from '../components/ConfirmDialog'
import DownloadGuideDialog from '../components/DownloadGuideDialog'

interface Certificate {
  id: number
  domain: string
  type: string
  expires_at: string | null
  cert_path: string
  key_path: string
  created_at: string
  proxy_hosts?: Array<{ id: number; domain: string; enabled: boolean }>
}

function getExpiryStatus(expiresAt: string | null): 'ok' | 'warning' | 'danger' | 'unknown' {
  if (!expiresAt) return 'unknown'
  const days = (new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
  if (days < 0) return 'danger'
  if (days < 7) return 'danger'
  if (days < 30) return 'warning'
  return 'ok'
}

function formatExpiry(expiresAt: string | null): string {
  if (!expiresAt) return '-'
  const days = Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
  const date = new Date(expiresAt).toLocaleDateString('en-US')
  if (days < 0) return `Expired (${date})`
  if (days === 0) return `Expires today`
  if (days === 1) return `Expires tomorrow`
  return `${date} (${days}d)`
}

export default function Certificates() {
  const [certs, setCerts] = useState<Certificate[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Certificate | null>(null)
  const [downloadTarget, setDownloadTarget] = useState<Certificate | null>(null)
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [page, setPage] = useState(1)

  const fetchCerts = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { page: String(page), limit: '20' }
      if (search) params.search = search
      if (typeFilter) params.type = typeFilter
      const { data } = await api.get('/certificates', { params })
      setCerts(data.data)
      setTotal(data.total)
    } finally {
      setLoading(false)
    }
  }, [page, search, typeFilter])

  useEffect(() => { fetchCerts() }, [fetchCerts])
  useEffect(() => { setPage(1) }, [search, typeFilter])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      const { data } = await api.delete(`/certificates/${deleteTarget.id}`)
      const parts = ['Deleted']
      if (data.detached_hosts > 0) {
        parts.push(`Detached from ${data.detached_hosts} proxy host(s)`)
      }
      parts.push('Files removed from server')
      toast.success(parts.join('. '), { duration: 5000 })
      setDeleteTarget(null)
      fetchCerts()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Delete failed')
    }
  }

  const handleRenew = async (id: number) => {
    try {
      await api.post(`/certificates/${id}/renew`)
      toast.success('Renewed')
      fetchCerts()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Renewal failed')
    }
  }

  const handleDownload = (cert: Certificate) => {
    setDownloadTarget(cert)
  }

  const typeBadges: Record<string, string> = {
    letsencrypt: 'bg-green-100 text-green-700',
    custom: 'bg-blue-100 text-blue-700',
    'self-signed': 'bg-purple-100 text-purple-700',
  }
  const typeLabels: Record<string, string> = {
    letsencrypt: "Let's Encrypt",
    custom: 'Custom',
    'self-signed': 'Self-Signed',
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Certificates</h2>
        <button
          onClick={() => setModalOpen(true)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus size={18} /> Add
        </button>
      </div>

      <TableToolbar
        search={search}
        onSearch={setSearch}
        searchPlaceholder="Search domains..."
        filters={[{
          key: 'type', value: typeFilter, label: 'Type',
          options: [
            { value: '', label: 'All Types' },
            { value: 'letsencrypt', label: "Let's Encrypt" },
            { value: 'custom', label: 'Custom' },
            { value: 'self-signed', label: 'Self-Signed' },
          ],
        }]}
        onFilter={(_, v) => setTypeFilter(v)}
      />

      {loading ? (
        <div className="bg-white rounded-xl shadow p-8 text-center text-gray-500">Loading...</div>
      ) : certs.length === 0 ? (
        <div className="bg-white rounded-xl shadow p-8 text-center">
          <p className="text-gray-500 mb-2">{search ? 'No results found.' : 'No certificates yet.'}</p>
          <p className="text-sm text-gray-400">Add an SSL certificate via Let's Encrypt or upload your own to enable HTTPS.</p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow overflow-x-auto">
          <table className="w-full text-sm min-w-[500px]">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="px-4 py-3 font-medium">Domain</th>
                <th className="px-4 py-3 font-medium">Type</th>
                <th className="px-4 py-3 font-medium">Used By</th>
                <th className="px-4 py-3 font-medium hidden sm:table-cell">Created</th>
                <th className="px-4 py-3 font-medium hidden md:table-cell">Expires</th>
                <th className="px-4 py-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {certs.map((cert) => {
                const status = getExpiryStatus(cert.expires_at)
                const hosts = cert.proxy_hosts || []
                const isExpanded = expandedId === cert.id
                return (
                  <>
                  <tr key={cert.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium">{cert.domain}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded text-xs ${typeBadges[cert.type] || 'bg-gray-100 text-gray-700'}`}>
                        {typeLabels[cert.type] || cert.type}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {hosts.length > 0 ? (
                        <button
                          onClick={() => setExpandedId(isExpanded ? null : cert.id)}
                          className="text-xs text-blue-600 hover:text-blue-700 hover:underline"
                        >
                          {hosts.length} host{hosts.length > 1 ? 's' : ''}
                        </button>
                      ) : (
                        <span className="text-xs text-gray-400">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-500 text-xs hidden sm:table-cell whitespace-nowrap">
                      {new Date(cert.created_at).toLocaleDateString('en-US', { month:'short', day:'numeric', year:'numeric' })}
                    </td>
                    <td className="px-4 py-3 hidden md:table-cell">
                      <span className={`flex items-center gap-1 ${
                        status === 'danger' ? 'text-red-600 font-medium' :
                        status === 'warning' ? 'text-yellow-600' : 'text-gray-600'
                      }`}>
                        {(status === 'danger' || status === 'warning') && <AlertTriangle size={14} />}
                        {formatExpiry(cert.expires_at)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        {cert.type === 'letsencrypt' && (
                          <button onClick={() => handleRenew(cert.id)} className="p-1 hover:text-green-600" title="Renew">
                            <RefreshCw size={16} />
                          </button>
                        )}
                        {cert.type === 'self-signed' && (
                          <button onClick={() => handleDownload(cert)} className="p-1 hover:text-purple-600" title="Download">
                            <Download size={16} />
                          </button>
                        )}
                        <button onClick={() => setDeleteTarget(cert)} className="p-1 hover:text-red-600" title="Delete">
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </td>
                  </tr>
                  {isExpanded && hosts.length > 0 && (
                    <tr className="bg-gray-50">
                      <td colSpan={6} className="px-4 py-3">
                        <p className="text-xs text-gray-400 mb-2">Used by:</p>
                        <div className="flex flex-wrap gap-1.5">
                          {hosts.map(h => (
                            <span key={h.id} className={`inline-flex items-center gap-1 px-2 py-1 rounded text-xs ${
                              h.enabled ? 'bg-blue-50 text-blue-700 border border-blue-200' : 'bg-gray-100 text-gray-500 border border-gray-200'
                            }`}>
                              {h.domain}
                              {!h.enabled && <span className="text-[10px]">(inactive)</span>}
                            </span>
                          ))}
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

      {modalOpen && <CertModal onClose={() => { setModalOpen(false); fetchCerts() }} />}

      {deleteTarget && (
        <ConfirmDialog
          title="Delete Certificate"
          message={`This will permanently delete the certificate for "${deleteTarget.domain}" from the server.`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}

      {downloadTarget && (
        <DownloadGuideDialog
          domain={downloadTarget.domain}
          certId={downloadTarget.id}
          onClose={() => setDownloadTarget(null)}
        />
      )}
    </div>
  )
}
