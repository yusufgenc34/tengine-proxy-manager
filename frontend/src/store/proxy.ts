import { create } from 'zustand'
import api from '../api/client'

export interface ProxyHost {
  id: number
  domain: string
  forward_host: string
  forward_port: number
  forward_scheme: string
  ssl_enabled: boolean
  health_check: boolean
  load_balancing: string
  enabled: boolean
  certificate_id: number | null
  access_list_id: number | null
  certificate?: any
  access_list?: any
  created_at: string
  updated_at: string
}

interface ProxyStore {
  hosts: ProxyHost[]
  total: number
  loading: boolean
  fetch: (params?: Record<string, string>) => Promise<void>
  create: (host: Partial<ProxyHost>) => Promise<void>
  update: (id: number, host: Partial<ProxyHost>) => Promise<void>
  remove: (id: number) => Promise<void>
  toggle: (id: number, enabled: boolean) => Promise<void>
}

export const useProxyStore = create<ProxyStore>((set) => ({
  hosts: [],
  total: 0,
  loading: false,

  fetch: async (params = {}) => {
    set({ loading: true })
    try {
      const { data } = await api.get('/proxy-hosts', { params })
      set({ hosts: data.data, total: data.total })
    } finally {
      set({ loading: false })
    }
  },

  create: async (host) => {
    await api.post('/proxy-hosts', host)
  },

  update: async (id, host) => {
    await api.put(`/proxy-hosts/${id}`, host)
  },

  remove: async (id) => {
    await api.delete(`/proxy-hosts/${id}`)
  },

  toggle: async (id, enabled) => {
    const action = enabled ? 'enable' : 'disable'
    await api.post(`/proxy-hosts/${id}/${action}`)
  },
}))
