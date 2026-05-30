import { X } from 'lucide-react'
import SearchInput from './SearchInput'

interface Filter {
  key: string
  value: string
  label: string
  options: { value: string; label: string }[]
}

interface Props {
  search: string
  onSearch: (value: string) => void
  searchPlaceholder: string
  filters?: Filter[]
  onFilter?: (key: string, value: string) => void
  children?: React.ReactNode
}

export default function TableToolbar({ search, onSearch, searchPlaceholder, filters, onFilter, children }: Props) {
  const activeFilters = filters?.filter(f => f.value) || []

  return (
    <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 mb-4">
      <div className="flex-1 sm:max-w-xs">
        <SearchInput value={search} onChange={onSearch} placeholder={searchPlaceholder} />
      </div>
      {filters?.map(f => (
        <select
          key={f.key}
          value={f.value}
          onChange={(e) => onFilter?.(f.key, e.target.value)}
          className="px-3 py-2 border rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {f.options.map(o => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      ))}
      {activeFilters.length > 0 && (
        <button
          onClick={() => activeFilters.forEach(f => onFilter?.(f.key, ''))}
          className="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded"
        >
          <X size={12} /> Clear filters
        </button>
      )}
      {children}
    </div>
  )
}
