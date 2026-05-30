import { useState, type FormEvent } from 'react'
import toast from 'react-hot-toast'
import api from '../api/client'
import Threads from '../components/Threads'
import DecryptedText from '../components/DecryptedText'

interface Props {
  onComplete: () => void
}

export default function Setup({ onComplete }: Props) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [saving, setSaving] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()

    if (password !== confirmPassword) {
      toast.error('Passwords do not match')
      return
    }

    if (password.length < 8) {
      toast.error('Password must be at least 8 characters')
      return
    }

    setSaving(true)
    try {
      await api.post('/setup', { email, password })
      toast.success('Admin account created! You can now log in.')
      onComplete()
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Setup failed'
      toast.error(msg)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-950 relative overflow-hidden">
      <div className="absolute inset-0">
        <Threads color={[0.4, 0.2, 1]} amplitude={1.5} distance={0.1} />
      </div>

      <div className="relative z-10 w-full max-w-md mx-4">
        <div className="text-center mb-10">
          <h1 className="text-4xl font-bold text-white tracking-tight">
            <DecryptedText
              text="Tengine Proxy Manager"
              animateOn="view"
              speed={40}
              maxIterations={15}
              sequential
              revealDirection="center"
              className="text-white"
              encryptedClassName="text-purple-400/60"
            />
          </h1>
          <p className="text-gray-400 mt-3 text-sm">Initial Setup</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-white/5 backdrop-blur-2xl border border-white/10 p-10 rounded-3xl shadow-2xl"
        >
          <p className="text-gray-400 text-sm mb-6 text-center">
            Create your admin account to get started.
          </p>
          <div className="mb-5">
            <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Email</label>
            <input
              type="email"
              placeholder="admin@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50 focus:bg-white/10 transition-all text-sm"
              required
            />
          </div>
          <div className="mb-5">
            <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Password</label>
            <input
              type="password"
              placeholder="Min 8 characters"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50 focus:bg-white/10 transition-all text-sm"
              required
              minLength={8}
            />
          </div>
          <div className="mb-8">
            <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Confirm Password</label>
            <input
              type="password"
              placeholder="Repeat password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50 focus:bg-white/10 transition-all text-sm"
              required
              minLength={8}
            />
          </div>
          <button
            type="submit"
            disabled={saving}
            className="w-full py-3.5 bg-purple-600 text-white font-medium rounded-xl hover:bg-purple-500 disabled:opacity-50 transition-all shadow-lg shadow-purple-600/20 text-sm tracking-wide"
          >
            {saving ? 'Creating...' : 'Create Admin Account'}
          </button>
        </form>
      </div>
    </div>
  )
}
