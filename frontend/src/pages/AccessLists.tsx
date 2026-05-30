import { useEffect, useState, useCallback, type FormEvent } from 'react'
import { Plus, Trash2, Pencil, PlusCircle, MinusCircle, ChevronDown, ChevronUp, ShieldCheck, ShieldX } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../api/client'
import TableToolbar from '../components/TableToolbar'
import Pagination from '../components/Pagination'
import ConfirmDialog from '../components/ConfirmDialog'
import Offcanvas from '../components/Offcanvas'

interface Rule {
  ip: string
  action: 'allow' | 'deny'
}

interface AccessList {
  id: number
  name: string
  rules: string
  created_at: string
}

export default function AccessLists() {
  const [lists, setLists] = useState<AccessList[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<AccessList | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AccessList | null>(null)
  const [name, setName] = useState('')
  const [rules, setRules] = useState<Rule[]>([{ ip: '', action: 'allow' }])
  const [mode, setMode] = useState<'simple' | 'expression'>('simple')
  const [expression, setExpression] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const fetchLists = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { page: String(page), limit: '20' }
      if (search) params.search = search
      const { data } = await api.get('/access-lists', { params })
      setLists(data.data)
      setTotal(data.total)
    } finally {
      setLoading(false)
    }
  }, [page, search])

  useEffect(() => { fetchLists() }, [fetchLists])
  useEffect(() => { setPage(1) }, [search])

  const rulesToExpression = (rulesStr: string) => {
    try {
      const parsed = JSON.parse(rulesStr || '[]')
      return parsed.map((r: Rule) => `${r.action} ${r.ip}`).join('\n')
    } catch {
      return ''
    }
  }

  const openModal = (list?: AccessList) => {
    if (list) {
      setEditing(list)
      setName(list.name)
      setMode('simple')
      try {
        const parsed = JSON.parse(list.rules || '[]')
        setRules(parsed.length > 0 ? parsed : [{ ip: '', action: 'allow' }])
        setExpression(rulesToExpression(list.rules))
      } catch {
        setRules([{ ip: '', action: 'allow' }])
        setExpression('')
      }
    } else {
      setEditing(null)
      setName('')
      setRules([{ ip: '', action: 'allow' }])
      setExpression('')
      setMode('simple')
    }
    setModalOpen(true)
  }

  const addRule = () => setRules([...rules, { ip: '', action: 'allow' }])

  const removeRule = (index: number) => {
    if (rules.length <= 1) return
    setRules(rules.filter((_, i) => i !== index))
  }

  const updateRule = (index: number, field: keyof Rule, value: string) => {
    const updated = [...rules]
    updated[index] = { ...updated[index], [field]: value }
    setRules(updated)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()

    let rulesJSON: string
    if (mode === 'expression') {
      try {
        const { data } = await api.post('/access-lists/parse', { expression })
        rulesJSON = data.rules
      } catch (err: any) {
        toast.error(err.response?.data?.message || 'Parse error')
        return
      }
    } else {
      const validRules = rules.filter(r => r.ip.trim() !== '')
      if (validRules.length === 0) {
        toast.error('At least one rule is required')
        return
      }
      rulesJSON = JSON.stringify(validRules)
    }

    const payload = { name, rules: rulesJSON }

    try {
      if (editing) {
        await api.put(`/access-lists/${editing.id}`, payload)
        toast.success('Updated')
      } else {
        await api.post('/access-lists', payload)
        toast.success('Created')
      }
      setModalOpen(false)
      fetchLists()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Operation failed')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.delete(`/access-lists/${deleteTarget.id}`)
      toast.success('Deleted')
      setDeleteTarget(null)
      fetchLists()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Delete failed')
    }
  }

  const parseRules = (rulesStr: string): Rule[] => {
    try { return JSON.parse(rulesStr || '[]') } catch { return [] }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Access Lists</h2>
        <button
          onClick={() => openModal()}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus size={18} /> Add
        </button>
      </div>

      <TableToolbar
        search={search}
        onSearch={setSearch}
        searchPlaceholder="Search access lists..."
      />

      {loading ? (
        <div className="bg-white rounded-xl shadow p-8 text-center text-gray-500">Loading...</div>
      ) : lists.length === 0 ? (
        <div className="bg-white rounded-xl shadow p-8 text-center">
          <p className="text-gray-500 mb-2">{search ? 'No results found.' : 'No access lists yet.'}</p>
          <p className="text-sm text-gray-400">
            {search ? 'Try a different search term.' : 'Access lists let you control which IP addresses can reach your proxy hosts.'}
          </p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow overflow-x-auto">
          <table className="w-full text-sm min-w-[500px]">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="px-4 py-3 font-medium w-8"></th>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Rules</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {lists.map((list) => {
                const listRules = parseRules(list.rules)
                const isExpanded = expandedId === list.id
                const allowCount = listRules.filter(r => r.action === 'allow').length
                const denyCount = listRules.filter(r => r.action === 'deny').length
                return (
                  <>
                    <tr key={list.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        {listRules.length > 0 && (
                          <button
                            onClick={() => setExpandedId(isExpanded ? null : list.id)}
                            className="p-0.5 hover:bg-gray-200 rounded"
                          >
                            {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                          </button>
                        )}
                      </td>
                      <td className="px-4 py-3 font-medium">{list.name}</td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          {listRules.length === 0 ? (
                            <span className="text-gray-400 text-xs">No rules</span>
                          ) : (
                            <>
                              {allowCount > 0 && (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-green-100 text-green-700">
                                  <ShieldCheck size={12} /> {allowCount} allow
                                </span>
                              )}
                              {denyCount > 0 && (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-red-100 text-red-700">
                                  <ShieldX size={12} /> {denyCount} deny
                                </span>
                              )}
                            </>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-gray-500">
                        {new Date(list.created_at).toLocaleDateString('en-US')}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button onClick={() => openModal(list)} className="p-1 hover:text-blue-600" title="Edit">
                            <Pencil size={16} />
                          </button>
                          <button onClick={() => setDeleteTarget(list)} className="p-1 hover:text-red-600" title="Delete">
                            <Trash2 size={16} />
                          </button>
                        </div>
                      </td>
                    </tr>
                    {isExpanded && listRules.length > 0 && (
                      <tr key={`${list.id}-rules`} className="bg-gray-50">
                        <td></td>
                        <td colSpan={4} className="px-4 py-3">
                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                            {listRules.map((rule, i) => (
                              <div
                                key={i}
                                className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm ${
                                  rule.action === 'allow'
                                    ? 'bg-green-50 border border-green-200'
                                    : 'bg-red-50 border border-red-200'
                                }`}
                              >
                                {rule.action === 'allow' ? (
                                  <ShieldCheck size={14} className="text-green-600" />
                                ) : (
                                  <ShieldX size={14} className="text-red-600" />
                                )}
                                <span className="font-mono text-xs">{rule.ip}</span>
                                <span className={`ml-auto text-xs font-medium ${
                                  rule.action === 'allow' ? 'text-green-600' : 'text-red-600'
                                }`}>
                                  {rule.action.toUpperCase()}
                                </span>
                              </div>
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

      {modalOpen && (
        <Offcanvas title={editing ? 'Edit Access List' : 'New Access List'} onClose={() => setModalOpen(false)}>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. Office Network"
                  className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-3">
                  <label className="block text-sm font-medium">Rules</label>
                  <div className="flex rounded-lg bg-gray-100 p-0.5">
                    <button
                      type="button"
                      onClick={() => setMode('simple')}
                      className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                        mode === 'simple' ? 'bg-white shadow text-gray-900' : 'text-gray-500 hover:text-gray-700'
                      }`}
                    >
                      Simple
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        if (mode === 'simple') {
                          const valid = rules.filter(r => r.ip.trim() !== '')
                          setExpression(valid.map(r => `${r.action} ${r.ip}`).join('\n'))
                        }
                        setMode('expression')
                      }}
                      className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                        mode === 'expression' ? 'bg-white shadow text-gray-900' : 'text-gray-500 hover:text-gray-700'
                      }`}
                    >
                      Expression
                    </button>
                  </div>
                </div>

                {mode === 'simple' ? (
                  <>
                    <div className="space-y-2">
                      {rules.map((rule, index) => (
                        <div key={index} className="flex items-center gap-2">
                          <input
                            type="text"
                            value={rule.ip}
                            onChange={(e) => updateRule(index, 'ip', e.target.value)}
                            placeholder="e.g. 192.168.1.0/24"
                            className="flex-1 px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                          <select
                            value={rule.action}
                            onChange={(e) => updateRule(index, 'action', e.target.value)}
                            className="px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          >
                            <option value="allow">Allow</option>
                            <option value="deny">Deny</option>
                          </select>
                          <button
                            type="button"
                            onClick={() => removeRule(index)}
                            className="p-1 text-gray-400 hover:text-red-500"
                            title="Remove rule"
                          >
                            <MinusCircle size={18} />
                          </button>
                        </div>
                      ))}
                    </div>
                    <div className="flex items-center gap-2 mt-2">
                      <button
                        type="button"
                        onClick={addRule}
                        className="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700"
                      >
                        <PlusCircle size={14} /> Add rule
                      </button>
                      <span className="text-xs text-gray-400">
                        Rules are evaluated top-to-bottom. Add deny all at the end for a whitelist.
                      </span>
                    </div>
                  </>
                ) : (
                  <div className="space-y-2">
                    <textarea
                      value={expression}
                      onChange={(e) => setExpression(e.target.value)}
                      placeholder={`allow 192.168.1.0/24\nallow 10.0.0.0/8\ndeny 0.0.0.0/0`}
                      rows={8}
                      className="w-full px-3 py-2 border rounded-lg font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                      spellCheck={false}
                    />
                    <details className="text-xs text-gray-400">
                      <summary className="cursor-pointer hover:text-gray-600">Syntax reference</summary>
                      <div className="mt-2 p-2 bg-gray-50 rounded space-y-0.5 font-mono">
                        <div><span className="text-green-600">allow</span> 192.168.1.1</div>
                        <div><span className="text-green-600">allow</span> 10.0.0.0/8</div>
                        <div><span className="text-red-600">deny</span> 172.16.0.0/12</div>
                        <div><span className="text-red-600">deny</span> 0.0.0.0/0</div>
                        <div className="text-gray-400 mt-1"># Lines starting with # are comments</div>
                      </div>
                    </details>
                  </div>
                )}
              </div>

              <div className="flex justify-end gap-2 border-t pt-4 mt-4">
                <button type="button" onClick={() => setModalOpen(false)} className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50">Cancel</button>
                <button type="submit" className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700">Save</button>
              </div>
            </form>
        </Offcanvas>
      )}

      {deleteTarget && (
        <ConfirmDialog
          title="Delete Access List"
          message={`Are you sure you want to delete "${deleteTarget.name}"? Proxy hosts using this list will lose their access restrictions.`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
