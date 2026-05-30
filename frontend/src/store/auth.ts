import { create } from 'zustand'
import api from '../api/client'

interface AuthState {
  isAuthenticated: boolean
  email: string | null
  loading: boolean
  twoFactorRequired: boolean
  tempToken: string | null
  login: (email: string, password: string) => Promise<void>
  verify2FA: (code: string) => Promise<void>
  logout: () => void
  checkAuth: () => void
  cancel2FA: () => void
}

function getEmailFromToken(): string | null {
  const token = localStorage.getItem('access_token')
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.email || null
  } catch {
    return null
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  isAuthenticated: !!localStorage.getItem('access_token'),
  email: getEmailFromToken(),
  loading: false,
  twoFactorRequired: false,
  tempToken: null,

  login: async (email, password) => {
    set({ loading: true })
    try {
      const { data } = await api.post('/auth/login', { email, password })
      if (data.two_factor_required) {
        set({ twoFactorRequired: true, tempToken: data.temp_token, email })
        return
      }
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      set({ isAuthenticated: true, email, twoFactorRequired: false, tempToken: null })
    } finally {
      set({ loading: false })
    }
  },

  verify2FA: async (code) => {
    set({ loading: true })
    try {
      const { tempToken, email } = get()
      const { data } = await api.post('/auth/2fa/login', {
        temp_token: tempToken,
        totp_code: code,
      })
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      set({ isAuthenticated: true, email, twoFactorRequired: false, tempToken: null })
    } finally {
      set({ loading: false })
    }
  },

  cancel2FA: () => {
    set({ twoFactorRequired: false, tempToken: null, email: null })
  },

  logout: () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    set({ isAuthenticated: false, email: null, twoFactorRequired: false, tempToken: null })
  },

  checkAuth: () => {
    set({ isAuthenticated: !!localStorage.getItem('access_token'), email: getEmailFromToken() })
  },
}))
