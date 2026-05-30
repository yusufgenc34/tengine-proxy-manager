import { ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

export default function CustomTemplates() {
  const navigate = useNavigate()

  return (
    <div className="max-w-3xl">
      <button onClick={() => navigate('/settings')} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4">
        <ArrowLeft size={14} /> Back to Settings
      </button>
      <h2 className="text-2xl font-bold mb-6">Custom Templates</h2>

      <div className="bg-white rounded-xl shadow p-8 text-center text-gray-400">
        Custom template management coming soon.
      </div>
    </div>
  )
}
