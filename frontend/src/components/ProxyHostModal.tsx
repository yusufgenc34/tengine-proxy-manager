import { useState, useEffect, type FormEvent } from 'react'
import toast from 'react-hot-toast'
import { useProxyStore, type ProxyHost } from '../store/proxy'
import api from '../api/client'
import Offcanvas from './Offcanvas'

interface Props {
  host: ProxyHost | null
  onClose: () => void
}

export default function ProxyHostModal({ host, onClose }: Props) {
  const { create, update } = useProxyStore()
  const isEdit = !!host

  const [domain, setDomain] = useState(host?.domain ?? '')
  const [forwardHost, setForwardHost] = useState(host?.forward_host ?? '')
  const [forwardPort, setForwardPort] = useState(host?.forward_port ?? 80)
  const [forwardScheme, setForwardScheme] = useState(host?.forward_scheme ?? 'http')
  const [sslEnabled, setSslEnabled] = useState(host?.ssl_enabled ?? false)
  const [healthCheck, setHealthCheck] = useState(host?.health_check ?? false)
  const [loadBalancing, setLoadBalancing] = useState(host?.load_balancing ?? '')
  const [certificateId, setCertificateId] = useState<number | null>(host?.certificate_id ?? null)
  const [accessListId, setAccessListId] = useState<number | null>(host?.access_list_id ?? null)
  const [certs, setCerts] = useState<Array<{id: number, domain: string}>>([])
  const [accessLists, setAccessLists] = useState<Array<{id: number, name: string}>>([])
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.get('/certificates', { params: { limit: '100' } }).then(({ data }) => setCerts(data.data || []))
    api.get('/access-lists', { params: { limit: '100' } }).then(({ data }) => setAccessLists(data.data || []))
  }, [])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const data = {
        domain,
        forward_host: forwardHost,
        forward_port: forwardPort,
        forward_scheme: forwardScheme,
        ssl_enabled: sslEnabled,
        health_check: healthCheck,
        load_balancing: loadBalancing,
        certificate_id: certificateId || null,
        access_list_id: accessListId || null,
      }

      if (isEdit) {
        await update(host.id, data)
        toast.success('Proxy host updated')
      } else {
        await create(data)
        toast.success('Proxy host created')
      }
      onClose()
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Operation failed'
      toast.error(msg)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Offcanvas title={isEdit ? 'Edit Proxy Host' : 'New Proxy Host'} onClose={onClose}>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1">Domain</label>
            <input
              type="text"
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              placeholder="e.g. app.example.com"
              className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
            <p className="text-xs text-gray-400 mt-1">The public domain that users will visit</p>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">Forward Host</label>
              <input
                type="text"
                value={forwardHost}
                onChange={(e) => setForwardHost(e.target.value)}
                placeholder="e.g. 10.0.0.1 or app-server"
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Forward Port</label>
              <input
                type="number"
                value={forwardPort}
                onChange={(e) => setForwardPort(Number(e.target.value))}
                placeholder="e.g. 3000"
                min={1}
                max={65535}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
            </div>
          </div>
          <p className="text-xs text-gray-400 -mt-1">Where traffic will be forwarded to (internal IP/hostname and port)</p>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">Scheme</label>
              <select
                value={forwardScheme}
                onChange={(e) => setForwardScheme(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
              </select>
              <p className="text-xs text-gray-400 mt-1">Protocol to connect to your backend</p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Load Balancing</label>
              <select
                value={loadBalancing}
                onChange={(e) => setLoadBalancing(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Round Robin</option>
                <option value="least_conn">Least Connections</option>
                <option value="consistent_hash">Consistent Hash</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">Certificate</label>
              <select
                value={certificateId ?? ''}
                onChange={(e) => setCertificateId(e.target.value ? Number(e.target.value) : null)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">None</option>
                {certs.map(cert => (
                  <option key={cert.id} value={cert.id}>{cert.domain}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Access List</label>
              <select
                value={accessListId ?? ''}
                onChange={(e) => setAccessListId(e.target.value ? Number(e.target.value) : null)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">None</option>
                {accessLists.map(al => (
                  <option key={al.id} value={al.id}>{al.name}</option>
                ))}
              </select>
            </div>
          </div>

          <div className="flex items-center gap-6 pt-1">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={sslEnabled}
                onChange={(e) => setSslEnabled(e.target.checked)}
                className="rounded"
              />
              Enable SSL
            </label>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={healthCheck}
                onChange={(e) => setHealthCheck(e.target.checked)}
                className="rounded"
              />
              Health Check
            </label>
          </div>
          <p className="text-xs text-gray-400 -mt-1">SSL requires a certificate. Health check monitors backend availability.</p>

          {sslEnabled && !certificateId && (
            <p className="text-xs text-yellow-600 bg-yellow-50 p-2 rounded">SSL is enabled but no certificate selected. HTTPS will not work without a certificate.</p>
          )}

          <div className="flex justify-end gap-2 border-t pt-4 mt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? 'Saving...' : isEdit ? 'Update' : 'Create'}
            </button>
          </div>
        </form>

        <div className="mt-8 bg-gray-50 rounded-lg p-4 text-xs text-gray-500 space-y-2">
          <p className="font-medium text-gray-600 text-sm">Load Balancing Methods</p>
          <p><span className="font-medium text-gray-700">Round Robin</span> — Distributes requests evenly across all servers in order. Default method.</p>
          <p><span className="font-medium text-gray-700">Least Connections</span> — Sends each request to the server with the fewest active connections. Best for long-running requests.</p>
          <p><span className="font-medium text-gray-700">Consistent Hash</span> — Routes requests based on URL, so the same URL always goes to the same server. Good for caching.</p>
        </div>

        <div className="mt-4 bg-gray-50 rounded-lg p-4 text-xs text-gray-500 space-y-2">
          <p className="font-medium text-gray-600 text-sm">SSL & Health Check</p>
          <p><span className="font-medium text-gray-700">SSL</span> — Enables HTTPS for this domain. Requires a certificate to be selected above.</p>
          <p><span className="font-medium text-gray-700">Health Check</span> — Tengine periodically checks if your backend is alive. If it goes down, traffic is paused until recovery.</p>
        </div>
    </Offcanvas>
  )
}
