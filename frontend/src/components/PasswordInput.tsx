import { useState } from 'react'
import { Eye, EyeOff, Sparkles, Copy, Check } from 'lucide-react'

interface Props {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  required?: boolean
  minLength?: number
  className?: string
  variant?: 'dark' | 'light'
  hideGenerate?: boolean
  onGenerate?: (pw: string) => void
}

const CHARS = {
  upper: 'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
  lower: 'abcdefghijklmnopqrstuvwxyz',
  digit: '0123456789',
  symbol: '!@#$%^&*()_+-=[]{}|;:,.<>?',
}

function generatePassword(len = 20): string {
  const all = CHARS.upper + CHARS.lower + CHARS.digit + CHARS.symbol
  const bytes = new Uint8Array(len)
  crypto.getRandomValues(bytes)

  const pool = [
    CHARS.upper[bytes[0] % CHARS.upper.length],
    CHARS.lower[bytes[1] % CHARS.lower.length],
    CHARS.digit[bytes[2] % CHARS.digit.length],
    CHARS.symbol[bytes[3] % CHARS.symbol.length],
  ]

  for (let i = 4; i < len; i++) {
    pool.push(all[bytes[i] % all.length])
  }

  for (let i = pool.length - 1; i > 0; i--) {
    const j = bytes[i] % (i + 1)
    ;[pool[i], pool[j]] = [pool[j], pool[i]]
  }

  return pool.join('')
}

function passwordStrength(pw: string): { label: string; color: string; pct: number } {
  if (!pw) return { label: '', color: 'bg-gray-200', pct: 0 }
  let score = 0
  if (pw.length >= 8) score++
  if (pw.length >= 14) score++
  if (/[A-Z]/.test(pw)) score++
  if (/[a-z]/.test(pw)) score++
  if (/[0-9]/.test(pw)) score++
  if (/[^A-Za-z0-9]/.test(pw)) score++

  if (score <= 2) return { label: 'Weak', color: 'bg-red-500', pct: 25 }
  if (score <= 4) return { label: 'Fair', color: 'bg-yellow-500', pct: 55 }
  return { label: 'Strong', color: 'bg-green-500', pct: 100 }
}

const darkInput = 'w-full px-5 py-3.5 pr-24 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:bg-white/10 transition-all text-sm'
const lightInput = 'w-full px-3 py-2 pr-20 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500'
const darkBtn = 'p-1.5 text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/10'
const lightBtn = 'p-1.5 text-gray-400 hover:text-gray-600 transition-colors rounded-lg hover:bg-gray-100'
const darkBar = 'flex-1 h-1.5 bg-white/10 rounded-full overflow-hidden'
const lightBar = 'flex-1 h-1.5 bg-gray-200 rounded-full overflow-hidden'

export default function PasswordInput({ value, onChange, placeholder = 'Password', required, minLength, className = '', variant = 'dark', hideGenerate, onGenerate }: Props) {
  const [show, setShow] = useState(false)
  const [copied, setCopied] = useState(false)

  const strength = passwordStrength(value)
  const d = variant === 'dark'

  const handleGenerate = () => {
    const pw = generatePassword(20)
    if (onGenerate) {
      onGenerate(pw)
    } else {
      onChange(pw)
    }
    setShow(true)
  }

  const handleCopy = async () => {
    if (!value) return
    await navigator.clipboard.writeText(value)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className={className}>
      <div className="relative">
        <input
          type={show ? 'text' : 'password'}
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={d ? darkInput : lightInput}
          required={required}
          minLength={minLength}
        />
        <div className="absolute right-1.5 top-1/2 -translate-y-1/2 flex items-center gap-0.5">
          {!hideGenerate && (
            <button type="button" onClick={handleGenerate} className={d ? darkBtn : lightBtn} title="Generate secure password">
              <Sparkles size={14} />
            </button>
          )}
          <button type="button" onClick={handleCopy} className={d ? darkBtn : lightBtn} title="Copy">
            {copied ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
          </button>
          <button type="button" onClick={() => setShow(!show)} className={d ? darkBtn : lightBtn} title={show ? 'Hide' : 'Show'}>
            {show ? <EyeOff size={14} /> : <Eye size={14} />}
          </button>
        </div>
      </div>

      {value && (
        <div className="mt-1.5 flex items-center gap-2">
          <div className={d ? darkBar : lightBar}>
            <div
              className={`h-full rounded-full transition-all duration-300 ${strength.color}`}
              style={{ width: `${strength.pct}%` }}
            />
          </div>
          <span className={`text-xs shrink-0 ${d ? 'text-gray-400' : 'text-gray-500'}`}>{strength.label}</span>
        </div>
      )}
    </div>
  )
}

export { generatePassword }
