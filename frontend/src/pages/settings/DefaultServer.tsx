import { useState, useEffect } from 'react'
import { ArrowLeft, Save } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import api from '../../api/client'

export default function DefaultServer() {
  const navigate = useNavigate()
  const [status, setStatus] = useState('444')
  const [customBody, setCustomBody] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.get('/settings/default-server').then(({ data }) => {
      setStatus(data.status || '444')
      setCustomBody(data.body || '')
    }).catch(() => {})
  }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      await api.put('/settings/default-server', { status, body: customBody })
      toast.success('Default server settings saved')
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="max-w-2xl">
      <button onClick={() => navigate('/settings')} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4">
        <ArrowLeft size={14} /> Back to Settings
      </button>
      <h2 className="text-2xl font-bold mb-6">Default Server</h2>

      <div className="bg-white rounded-xl shadow p-6 space-y-5">
        <div>
          <label className="block text-sm font-medium mb-2">Response Status</label>
          <select value={status} onChange={(e) => setStatus(e.target.value)}
            className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option value="444">444 — Close connection (no response)</option>
            <option value="200">200 — Return custom page</option>
            <option value="301">301 — Redirect</option>
            <option value="403">403 — Forbidden</option>
            <option value="404">404 — Not Found</option>
            <option value="502">502 — Bad Gateway</option>
          </select>
          <p className="text-xs text-gray-400 mt-1">
            This is the response for requests that don't match any proxy host. Default is 444 (silent drop).
          </p>
        </div>

        {status === '200' && (
          <div>
            <label className="block text-sm font-medium mb-2">Custom Response Body (HTML)</label>
            <textarea value={customBody} onChange={(e) => setCustomBody(e.target.value)}
              rows={8} placeholder="<html><body>...</body></html>"
              className="w-full px-3 py-2 border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none" />
          </div>
        )}

        {status === '301' && (
          <div>
            <label className="block text-sm font-medium mb-2">Redirect URL</label>
            <input type="text" value={customBody} onChange={(e) => setCustomBody(e.target.value)}
              placeholder="https://example.com"
              className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        )}

        <div className="flex justify-end border-t pt-4">
          <button onClick={handleSave} disabled={saving}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 disabled:opacity-50">
            <Save size={16} /> {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
