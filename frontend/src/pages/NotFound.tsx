import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Home } from 'lucide-react'

export default function NotFound() {
  const navigate = useNavigate()

  return (
    <div className="relative flex items-center justify-center" style={{ height: 'calc(100vh - 112px)' }}>

      {/* Grid background */}
      <div
        className="absolute inset-0 opacity-40 pointer-events-none z-0"
        style={{
          backgroundImage:
            'linear-gradient(rgb(0 0 0 / 0.06) 1px, transparent 1px), linear-gradient(90deg, rgb(0 0 0 / 0.06) 1px, transparent 1px)',
          backgroundSize: '40px 40px',
        }}
      />

      {/* Corner accents */}
      <div className="absolute top-5 left-5 w-16 h-16 border-t border-l border-gray-300 pointer-events-none z-0" />
      <div className="absolute bottom-5 right-5 w-16 h-16 border-b border-r border-gray-300 pointer-events-none z-0" />

      {/* Content */}
      <div className="relative z-10 text-center">

        {/* 404 large text */}
        <div className="relative inline-block leading-none select-none mb-14">
          <span
            className="absolute inset-0 font-mono font-bold text-[clamp(80px,18vw,140px)] tracking-tighter"
            style={{
              color: 'transparent',
              WebkitTextStroke: '1.5px rgb(156 163 175)',
              transform: 'translate(4px, 4px)',
              userSelect: 'none',
            }}
            aria-hidden
          >
            404
          </span>
          <span className="relative font-mono font-bold text-[clamp(80px,18vw,140px)] tracking-tighter text-gray-900">
            404
          </span>
        </div> 

        <h2 className="text-xl font-semibold text-gray-900 mb-2">Something went missing</h2>
        <p className="text-sm text-gray-500 max-w-xs mx-auto leading-relaxed mb-8">
          The page you're looking for may have been moved, deleted, or never existed.
        </p>

        {/* Buttons */}
        <div className="flex items-center justify-center gap-2.5 flex-wrap relative z-20">
          <button
            onClick={() => window.history.length > 1 ? navigate(-1) : navigate('/')}
            className="inline-flex items-center gap-1.5 px-4 py-2 text-sm text-gray-500 hover:text-gray-700 bg-white hover:bg-gray-50 border border-gray-200 rounded-lg transition-colors"
          >
            <ArrowLeft size={15} />
            Go back
          </button>
          <button
            onClick={() => navigate('/')}
            className="inline-flex items-center gap-1.5 px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
          >
            <Home size={15} />
            Dashboard
          </button>
        </div>
      </div>
    </div>
  )
}
