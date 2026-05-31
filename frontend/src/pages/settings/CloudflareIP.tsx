import { useState, useEffect } from 'react'
import { ArrowLeft, Shield, RefreshCw } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import api from '../../api/client'

interface CloudflareState {
  enabled: boolean
  ipv4_count: number
  ipv6_count: number
  last_fetched: string
  updated_at: string
}

export default function CloudflareIP() {
  const navigate = useNavigate()
  const [cfg, setCfg] = useState<CloudflareState | null>(null)
  const [loading, setLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)

  useEffect(() => {
    api.get('/settings/cloudflare')
      .then(({ data }) => setCfg(data))
      .catch(() => toast.error('Failed to load Cloudflare settings'))
  }, [])

  const handleToggle = async () => {
    if (!cfg) return
    setLoading(true)
    try {
      const { data } = await api.put('/settings/cloudflare', { enabled: !cfg.enabled })
      setCfg(data)
      toast.success(data.enabled ? 'Cloudflare IP whitelist enabled' : 'Cloudflare IP whitelist disabled')
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Failed to update')
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      const { data } = await api.post('/settings/cloudflare/refresh')
      setCfg(data)
      toast.success(`IP list refreshed (v4: ${data.ipv4_count}, v6: ${data.ipv6_count})`)
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Refresh failed')
    } finally {
      setRefreshing(false)
    }
  }

  const formatDate = (s: string) => {
    if (!s) return 'Never'
    return new Date(s).toLocaleString()
  }

  return (
    <div className="max-w-2xl">
      <button onClick={() => navigate('/settings')} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4">
        <ArrowLeft size={14} /> Back to Settings
      </button>
      <h2 className="text-2xl font-bold mb-6">Cloudflare IP Whitelist</h2>

      <div className="bg-white rounded-xl shadow p-6 space-y-6">
        {/* Toggle */}
        <div className="flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-sm">Allow Cloudflare IPs Only</h3>
            <p className="text-xs text-gray-400 mt-1">
              When enabled, only Cloudflare IP ranges can access ports 80 and 443.
              All other traffic is denied at the proxy level.
            </p>
          </div>
          <button
            onClick={handleToggle}
            disabled={loading}
            className={`inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:opacity-50 ${
              cfg?.enabled ? 'bg-orange-500' : 'bg-gray-300'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 rounded-full bg-white shadow transition-transform ${
                cfg?.enabled ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
        </div>

        {cfg?.enabled && (
          <>
            <div className="border-t pt-4 space-y-3">
              {/* Stats */}
              <div className="grid grid-cols-2 gap-4">
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 text-center">
                  <div className="text-2xl font-bold text-gray-800">{cfg.ipv4_count}</div>
                  <div className="text-xs text-gray-500">IPv4 Ranges</div>
                </div>
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 text-center">
                  <div className="text-2xl font-bold text-gray-800">{cfg.ipv6_count}</div>
                  <div className="text-xs text-gray-500">IPv6 Ranges</div>
                </div>
              </div>

              {/* Last fetched */}
              <div className="flex items-center justify-between text-sm text-gray-500">
                <span>Last fetched: {formatDate(cfg.last_fetched)}</span>
                <button
                  onClick={handleRefresh}
                  disabled={refreshing}
                  className="flex items-center gap-1 px-3 py-1.5 bg-gray-100 rounded-lg hover:bg-gray-200 disabled:opacity-50 text-xs font-medium"
                >
                  <RefreshCw size={14} className={refreshing ? 'animate-spin' : ''} />
                  {refreshing ? 'Refreshing...' : 'Refresh Now'}
                </button>
              </div>
            </div>

            {/* Info */}
            <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 flex gap-3">
              <Shield size={18} className="text-amber-600 shrink-0 mt-0.5" />
              <div className="text-xs text-amber-700 space-y-1">
                <p className="font-semibold">How it works:</p>
                <ul className="list-disc list-inside space-y-0.5">
                  <li>Cloudflare IP ranges are fetched from cloudflare.com/ips-v4 and /ips-v6</li>
                  <li>The list auto-refreshes every 7 days</li>
                  <li>Local/private IPs (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) are always allowed</li>
                  <li>Tengine reloads automatically when the list updates</li>
                </ul>
              </div>
            </div>
          </>
        )}

        {!cfg?.enabled && (
          <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
            <p className="text-xs text-gray-500">
              Enable this feature to restrict access to only Cloudflare IP addresses.
              This is useful when your domains are proxied through Cloudflare and you want to
              prevent direct server access bypassing Cloudflare.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
