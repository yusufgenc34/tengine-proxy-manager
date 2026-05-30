import { useEffect, useState, useCallback, type FormEvent } from 'react'
import { Plus, Trash2, Pencil } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../api/client'
import TableToolbar from '../components/TableToolbar'
import Pagination from '../components/Pagination'
import ConfirmDialog from '../components/ConfirmDialog'
import Offcanvas from '../components/Offcanvas'

interface User {
  id: number
  email: string
  role: string
  created_at: string
}

export default function Users() {
  const [users, setUsers] = useState<User[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('user')
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState('')
  const [page, setPage] = useState(1)

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { page: String(page), limit: '20' }
      if (search) params.search = search
      if (roleFilter) params.role = roleFilter
      const { data } = await api.get('/users', { params })
      setUsers(data.data)
      setTotal(data.total)
    } finally {
      setLoading(false)
    }
  }, [page, search, roleFilter])

  useEffect(() => { fetchUsers() }, [fetchUsers])
  useEffect(() => { setPage(1) }, [search, roleFilter])

  const openModal = (user?: User) => {
    if (user) {
      setEditing(user)
      setEmail(user.email)
      setRole(user.role)
      setPassword('')
    } else {
      setEditing(null)
      setEmail('')
      setPassword('')
      setRole('user')
    }
    setModalOpen(true)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    try {
      if (editing) {
        const body: Record<string, string> = { email, role }
        if (password) body.password = password
        await api.put(`/users/${editing.id}`, body)
        toast.success('Updated')
      } else {
        await api.post('/users', { email, password, role })
        toast.success('Created')
      }
      setModalOpen(false)
      fetchUsers()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Operation failed')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.delete(`/users/${deleteTarget.id}`)
      toast.success('Deleted')
      setDeleteTarget(null)
      fetchUsers()
    } catch (err: any) {
      toast.error(err.response?.data?.message || 'Delete failed')
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Users</h2>
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
        searchPlaceholder="Search users..."
        filters={[{
          key: 'role', value: roleFilter, label: 'Role',
          options: [{ value: '', label: 'All Roles' }, { value: 'admin', label: 'Admin' }, { value: 'user', label: 'User' }],
        }]}
        onFilter={(_, v) => setRoleFilter(v)}
      />

      {loading ? (
        <div className="bg-white rounded-xl shadow p-8 text-center text-gray-500">Loading...</div>
      ) : users.length === 0 ? (
        <div className="bg-white rounded-xl shadow p-8 text-center">
          <p className="text-gray-500 mb-2">{search ? 'No results found.' : 'No users yet.'}</p>
          <p className="text-sm text-gray-400">
            {search ? 'Try a different search term.' : 'Create user accounts to manage who can access this panel.'}
          </p>
        </div>
      ) : (
        <div className="bg-white rounded-xl shadow overflow-x-auto">
          <table className="w-full text-sm min-w-[400px]">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="px-4 py-3 font-medium">Email</th>
                <th className="px-4 py-3 font-medium">Role</th>
                <th className="px-4 py-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium">{user.email}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded text-xs ${user.role === 'admin' ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-600'}`}>
                      {user.role}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button onClick={() => openModal(user)} className="p-1 hover:text-blue-600" title="Edit">
                        <Pencil size={16} />
                      </button>
                      <button onClick={() => setDeleteTarget(user)} className="p-1 hover:text-red-600" title="Delete">
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pagination page={page} limit={20} total={total} onPageChange={setPage} />
        </div>
      )}

      {modalOpen && (
        <Offcanvas title={editing ? 'Edit User' : 'New User'} onClose={() => setModalOpen(false)}>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">Email</label>
                <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" required />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Password {editing && '(leave blank to keep)'}</label>
                <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" required={!editing} />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Role</label>
                <select value={role} onChange={(e) => setRole(e.target.value)} className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">
                  <option value="user">User</option>
                  <option value="admin">Admin</option>
                </select>
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
          title="Delete User"
          message={`Are you sure you want to delete "${deleteTarget.email}"? This action cannot be undone.`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
