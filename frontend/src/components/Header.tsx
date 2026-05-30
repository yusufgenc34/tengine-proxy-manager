import { LogOut, User, Menu } from 'lucide-react'
import { useAuthStore } from '../store/auth'

interface Props {
  onMenuClick: () => void
}

export default function Header({ onMenuClick }: Props) {
  const { email, logout } = useAuthStore()

  return (
    <header className="h-14 bg-white border-b border-gray-200 flex items-center justify-between px-4 lg:px-6 shrink-0">
      <button
        onClick={onMenuClick}
        className="p-1.5 hover:bg-gray-100 rounded-lg lg:hidden"
        aria-label="Toggle menu"
      >
        <Menu size={20} />
      </button>
      <div className="hidden lg:block" />
      <div className="flex items-center gap-3">
        {email && (
          <div className="hidden sm:flex items-center gap-2 text-sm text-gray-600">
            <User size={16} />
            <span className="truncate max-w-[160px]">{email}</span>
          </div>
        )}
        <button
          onClick={logout}
          className="flex items-center gap-2 text-sm text-gray-600 hover:text-red-600 transition-colors"
        >
          <LogOut size={16} />
          <span className="hidden sm:inline">Logout</span>
        </button>
      </div>
    </header>
  )
}
