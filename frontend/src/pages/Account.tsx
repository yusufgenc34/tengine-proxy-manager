import { useState, useEffect, type FormEvent } from 'react'
import { ShieldCheck, ShieldOff, Copy, Check, KeyRound, ChevronDown } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../api/client'
import { useAuthStore } from '../store/auth'
import PasswordInput from '../components/PasswordInput'

export default function Account() {
  const { checkAuth } = useAuthStore()
  const [passwordOpen, setPasswordOpen] = useState(false)
  const [twofaOpen, setTwofaOpen] = useState(false)
  const [enabled, setEnabled] = useState(false)
  const [loading, setLoading] = useState(true)
  const [setupData, setSetupData] = useState<{ secret: string; qr_code: string } | null>(null)
  const [verifyCode, setVerifyCode] = useState('')
  const [disableCode, setDisableCode] = useState('')
  const [saving, setSaving] = useState(false)
  const [copied, setCopied] = useState(false)
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [pwSaving, setPwSaving] = useState(false)

  const fetchStatus = async () => {
    setLoading(true)
    try {
      const { data } = await api.get('/auth/2fa/status')
      setEnabled(data.two_factor_enabled)
    } finally { setLoading(false) }
  }

  useEffect(() => { fetchStatus() }, [])

  const handleSetup = async () => {
    try { const { data } = await api.post('/auth/2fa/setup'); setSetupData(data) }
    catch (err: any) { toast.error(err.response?.data?.message || 'Failed to setup 2FA') }
  }

  const handleVerify = async (e: FormEvent) => {
    e.preventDefault(); setSaving(true)
    try { await api.post('/auth/2fa/verify', { code: verifyCode }); toast.success('2FA enabled'); setEnabled(true); setSetupData(null); setVerifyCode('') }
    catch (err: any) { toast.error(err.response?.data?.message || 'Invalid code'); setVerifyCode('') }
    finally { setSaving(false) }
  }

  const handleDisable = async (e: FormEvent) => {
    e.preventDefault(); setSaving(true)
    try { await api.post('/auth/2fa/disable', { code: disableCode }); toast.success('2FA disabled'); setEnabled(false); setDisableCode('') }
    catch (err: any) { toast.error(err.response?.data?.message || 'Invalid code'); setDisableCode('') }
    finally { setSaving(false) }
  }

  const handleGeneratePassword = (pw: string) => {
    setNewPassword(pw)
    setConfirmPassword(pw)
  }

  const handlePasswordChange = async (e: FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) { toast.error('Passwords do not match'); return }
    if (newPassword.length < 8) { toast.error('Password must be at least 8 characters'); return }
    setPwSaving(true)
    try { await api.put('/auth/password', { current_password: currentPassword, new_password: newPassword }); toast.success('Password changed'); setCurrentPassword(''); setNewPassword(''); setConfirmPassword(''); setPasswordOpen(false) }
    catch (err: any) { toast.error(err.response?.data?.message || 'Failed to change password') }
    finally { setPwSaving(false) }
  }

  if (loading) return <div className="bg-white rounded-xl shadow p-8 text-center text-gray-500">Loading...</div>

  return (
    <div className="max-w-2xl">
      <h2 className="text-2xl font-bold mb-6">Account</h2>
      <div className="space-y-3">
        {/* Password */}
        <div className="bg-white rounded-xl shadow overflow-hidden">
          <button onClick={() => setPasswordOpen(!passwordOpen)} className="w-full px-5 py-4 flex items-center gap-3 hover:bg-gray-50/50 transition-colors text-left">
            <div className="w-9 h-9 bg-blue-100 rounded-lg flex items-center justify-center shrink-0"><KeyRound size={18} className="text-blue-600" /></div>
            <div className="flex-1 min-w-0"><h3 className="font-semibold text-sm">Change Password</h3><p className="text-xs text-gray-400">Update your account password</p></div>
            <ChevronDown size={18} className={`text-gray-400 shrink-0 transition-transform ${passwordOpen ? 'rotate-180' : ''}`} />
          </button>
          {passwordOpen && (
            <form onSubmit={handlePasswordChange} className="px-5 pb-5 space-y-3 border-t">
              <div className="pt-4"><label className="block text-xs font-medium text-gray-500 mb-1">Current Password</label><input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" required /></div>
              <div className="grid grid-cols-2 gap-3"><div><label className="block text-xs font-medium text-gray-500 mb-1">New Password</label><PasswordInput value={newPassword} onChange={setNewPassword} onGenerate={handleGeneratePassword} variant="light" minLength={8} required /></div><div><label className="block text-xs font-medium text-gray-500 mb-1">Confirm</label><PasswordInput value={confirmPassword} onChange={setConfirmPassword} variant="light" minLength={8} required hideGenerate /></div></div>
              <div className="flex justify-end pt-1"><button type="submit" disabled={pwSaving} className="px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 disabled:opacity-50">{pwSaving ? 'Changing...' : 'Change Password'}</button></div>
            </form>
          )}
        </div>

        {/* 2FA */}
        <div className="bg-white rounded-xl shadow overflow-hidden">
          <button onClick={() => setTwofaOpen(!twofaOpen)} className="w-full px-5 py-4 flex items-center gap-3 hover:bg-gray-50/50 transition-colors text-left">
            {enabled ? <div className="w-9 h-9 bg-green-100 rounded-lg flex items-center justify-center shrink-0"><ShieldCheck size={18} className="text-green-600" /></div> : <div className="w-9 h-9 bg-gray-100 rounded-lg flex items-center justify-center shrink-0"><ShieldOff size={18} className="text-gray-400" /></div>}
            <div className="flex-1 min-w-0"><h3 className="font-semibold text-sm">Two-Factor Authentication</h3><p className="text-xs text-gray-400">{enabled ? 'Your account is protected with 2FA' : 'Add an extra layer of security'}</p></div>
            <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium shrink-0 ${enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>{enabled ? 'On' : 'Off'}</span>
            <ChevronDown size={18} className={`text-gray-400 shrink-0 transition-transform ${twofaOpen ? 'rotate-180' : ''}`} />
          </button>
          {twofaOpen && (
            <div className="px-5 pb-5 border-t">
              <div className="pt-4">
                {enabled && !setupData && (
                  <form onSubmit={handleDisable} className="space-y-3">
                    <p className="text-sm text-gray-500">Enter a code from your authenticator app to disable 2FA.</p>
                    <div className="flex gap-2"><input type="text" inputMode="numeric" maxLength={6} placeholder="000000" value={disableCode} onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))} className="flex-1 px-4 py-2 border rounded-lg font-mono text-center tracking-[0.25em] text-sm focus:outline-none focus:ring-2 focus:ring-red-500" required /><button type="submit" disabled={saving || disableCode.length !== 6} className="px-4 py-2 bg-red-600 text-white text-sm rounded-lg hover:bg-red-700 disabled:opacity-50 whitespace-nowrap">{saving ? '...' : 'Disable'}</button></div>
                  </form>
                )}
                {!enabled && !setupData && (
                  <div className="space-y-4"><p className="text-sm text-gray-500">Use an authenticator app like Google Authenticator, Authy, or 1Password to generate one-time codes.</p><button onClick={handleSetup} className="px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700">Set Up 2FA</button></div>
                )}
                {setupData && (
                  <div className="space-y-4">
                    <p className="text-sm text-gray-500">Scan the QR code with your authenticator app, then enter the verification code.</p>
                    <div className="flex items-start gap-4"><div className="bg-white p-2 rounded-lg border shadow-sm shrink-0"><img src={setupData.qr_code} alt="2FA QR Code" className="w-36 h-36" /></div><div className="flex-1 min-w-0"><p className="text-xs text-gray-400 mb-1.5">Or enter key manually:</p><div className="flex items-center gap-1.5"><code className="text-xs bg-gray-100 px-2 py-1.5 rounded font-mono truncate select-all block">{setupData.secret}</code><button onClick={() => { navigator.clipboard.writeText(setupData.secret); setCopied(true); setTimeout(() => setCopied(false), 2000) }} className="p-1 hover:bg-gray-100 rounded shrink-0">{copied ? <Check size={14} className="text-green-600" /> : <Copy size={14} className="text-gray-400" />}</button></div></div></div>
                    <form onSubmit={handleVerify} className="space-y-3"><div className="flex gap-2"><input type="text" inputMode="numeric" maxLength={6} placeholder="000000" value={verifyCode} onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, '').slice(0, 6))} className="flex-1 px-4 py-2 border rounded-lg font-mono text-center tracking-[0.25em] text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" autoFocus required /><button type="submit" disabled={saving || verifyCode.length !== 6} className="px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 disabled:opacity-50 whitespace-nowrap">{saving ? '...' : 'Verify & Enable'}</button></div><button type="button" onClick={() => { setSetupData(null); setVerifyCode('') }} className="text-xs text-gray-400 hover:text-gray-600">Cancel setup</button></form>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
