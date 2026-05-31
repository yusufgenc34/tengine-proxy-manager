import { useNavigate } from 'react-router-dom'
import { Globe, Shield, ArrowRight } from 'lucide-react'

const modules = [
  {
    id: 'default-server',
    title: 'Default Server',
    desc: 'Configure what Tengine returns on unmatched domains and port 80 fallback.',
    icon: Globe,
    color: 'bg-blue-500',
  },
  {
    id: 'cloudflare',
    title: 'Cloudflare IP Whitelist',
    desc: 'Restrict ports 80 & 443 to only Cloudflare IP ranges. Auto-updates weekly.',
    icon: Shield,
    color: 'bg-orange-500',
  },
]

export default function Settings() {
  const navigate = useNavigate()

  return (
    <div className="max-w-4xl">
      <h2 className="text-2xl font-bold mb-6">Settings</h2>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {modules.map(({ id, title, desc, icon: Icon, color }) => (
          <button
            key={id}
            onClick={() => navigate(`/settings/${id}`)}
            className="bg-white rounded-xl shadow p-5 flex items-start gap-4 text-left hover:shadow-md transition-shadow group"
          >
            <div className={`${color} p-3 rounded-lg text-white shrink-0`}>
              <Icon size={22} />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="font-semibold text-sm group-hover:text-blue-600 transition-colors">{title}</h3>
              <p className="text-xs text-gray-400 mt-1">{desc}</p>
            </div>
            <ArrowRight size={16} className="text-gray-300 group-hover:text-blue-500 shrink-0 mt-1 transition-colors" />
          </button>
        ))}
      </div>
    </div>
  )
}
