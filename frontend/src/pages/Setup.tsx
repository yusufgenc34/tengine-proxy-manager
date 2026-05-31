import { useState, type FormEvent } from 'react'
import toast from 'react-hot-toast'
import api from '../api/client'
import Threads from '../components/Threads'
import DecryptedText from '../components/DecryptedText'
import PasswordInput from '../components/PasswordInput'

interface Props {
  onComplete: () => void
}

export default function Setup({ onComplete }: Props) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [setupKey, setSetupKey] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const handleGenerate = (pw: string) => {
    setPassword(pw)
    setConfirmPassword(pw)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()

    setError('')

    if (!setupKey.trim()) {
      setError('Setup key is required')
      return
    }

    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }

    setSaving(true)
    try {
      await api.post('/setup', { email, password, setup_key: setupKey.trim() })
      toast.success('Admin account created! You can now log in.')
      onComplete()
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Setup failed'
      setError(msg)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-950 relative overflow-hidden">
      <div className="absolute inset-0">
        <Threads color={[0.1, 0.7, 0.7]} amplitude={1.5} distance={0.1} />
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
              encryptedClassName="text-cyan-400/60"
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
              className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:bg-white/10 transition-all text-sm"
              required
            />
          </div>
          <div className="mb-5">
            <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Password</label>
            <PasswordInput
              value={password}
              onChange={setPassword}
              onGenerate={handleGenerate}
              placeholder="Min 8 characters"
              required
              minLength={8}
            />
          </div>
          <div className="mb-8">
            <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Confirm Password</label>
            <PasswordInput
              value={confirmPassword}
              onChange={setConfirmPassword}
              placeholder="Repeat password"
              required
              minLength={8}
              hideGenerate
            />
       </div>
      <div className="mb-5">
        <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">
          Setup Key
        </label>
        <input
          type="text"
          placeholder="Enter setup key"
          value={setupKey}
          onChange={(e) => setSetupKey(e.target.value)}
          className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:bg-white/10 transition-all text-sm font-mono"
          required
        />
        <p className="mt-2 flex items-start gap-1.5 text-xs text-gray-500 font-light leading-relaxed">
          <span className="mt-px text-gray-600">*</span>
          <span>
            You can retrieve your setup key by running{" "}
            <code className="font-mono text-gray-400 bg-white/5 px-1 py-0.5 rounded">
              docker exec tengineproxymanager-backend-1 cat /app/setup.key
            </code>{" "}
            in your terminal.
          </span>
        </p>
      </div>
          {error && (
            <div className="mb-5 px-4 py-3 bg-red-500/10 border border-red-500/30 rounded-xl text-red-400 text-sm text-center">
              {error}
            </div>
          )}
          <button
            type="submit"
            disabled={saving}
            className="w-full py-3.5 bg-cyan-600 text-white font-medium rounded-xl hover:bg-cyan-500 disabled:opacity-50 transition-all shadow-lg shadow-cyan-600/20 text-sm tracking-wide"
          >
            {saving ? 'Creating...' : 'Create Admin Account'}
          </button>
        </form>
      </div>
    </div>
  )
}
