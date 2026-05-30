import { useState, useRef, useEffect, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import { useAuthStore } from '../store/auth'
import Threads from '../components/Threads'
import DecryptedText from '../components/DecryptedText'

export default function Login() {
  const { isAuthenticated, login, verify2FA, cancel2FA, loading, twoFactorRequired } = useAuthStore()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const totpInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (twoFactorRequired && totpInputRef.current) {
      totpInputRef.current.focus()
    }
  }, [twoFactorRequired])

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await login(email, password)
    } catch {
      toast.error('Invalid email or password')
    }
  }

  const handle2FA = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await verify2FA(totpCode)
      toast.success('Login successful')
    } catch {
      toast.error('Invalid 2FA code')
      setTotpCode('')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-950 relative overflow-hidden">
      <div className="absolute inset-0">
        <Threads color={[0.2, 0.4, 1]} amplitude={1.2} distance={0} />
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
              encryptedClassName="text-blue-400/60"
            />
          </h1>
        </div>

        {!twoFactorRequired ? (
          <form
            onSubmit={handleLogin}
            className="bg-white/5 backdrop-blur-2xl border border-white/10 p-10 rounded-3xl shadow-2xl"
          >
            <div className="mb-5">
              <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Email</label>
              <input
                type="email"
                placeholder="admin@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:bg-white/10 transition-all text-sm"
                required
              />
            </div>
            <div className="mb-8">
              <label className="block text-xs font-medium text-gray-400 mb-2 uppercase tracking-wider">Password</label>
              <input
                type="password"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-5 py-3.5 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:bg-white/10 transition-all text-sm"
                required
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3.5 bg-blue-600 text-white font-medium rounded-xl hover:bg-blue-500 disabled:opacity-50 transition-all shadow-lg shadow-blue-600/20 text-sm tracking-wide"
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
        ) : (
          <form
            onSubmit={handle2FA}
            className="bg-white/5 backdrop-blur-2xl border border-white/10 p-10 rounded-3xl shadow-2xl"
          >
            <div className="text-center mb-6">
              <div className="w-14 h-14 bg-blue-500/20 rounded-2xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              <p className="text-sm text-gray-400">Enter the 6-digit code from your authenticator app</p>
            </div>
            <div className="mb-6">
              <input
                ref={totpInputRef}
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                placeholder="000000"
                value={totpCode}
                onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                className="w-full px-5 py-4 bg-white/5 border border-white/10 rounded-xl text-white text-center text-2xl tracking-[0.5em] font-mono placeholder-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:bg-white/10 transition-all"
                required
              />
            </div>
            <button
              type="submit"
              disabled={loading || totpCode.length !== 6}
              className="w-full py-3.5 bg-blue-600 text-white font-medium rounded-xl hover:bg-blue-500 disabled:opacity-50 transition-all shadow-lg shadow-blue-600/20 text-sm tracking-wide"
            >
              {loading ? 'Verifying...' : 'Verify'}
            </button>
            <button
              type="button"
              onClick={() => { cancel2FA(); setTotpCode('') }}
              className="w-full mt-3 py-2.5 text-gray-400 text-sm hover:text-white transition-colors"
            >
              Back to login
            </button>
          </form>
        )}
      </div>
    </div>
  )
}
