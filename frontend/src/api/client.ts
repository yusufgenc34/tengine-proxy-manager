import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshPromise: Promise<string> | null = null

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry && !['/auth/login', '/auth/refresh', '/auth/2fa/login'].includes(originalRequest.url ?? '')) {
      originalRequest._retry = true

      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken) {
        try {
          if (!refreshPromise) {
            refreshPromise = axios.post('/api/v1/auth/refresh', { refresh_token: refreshToken }, { timeout: 15000 })
              .then(({ data }) => {
                localStorage.setItem('access_token', data.access_token)
                return data.access_token as string
              }).finally(() => { refreshPromise = null })
          }
          const accessToken = await refreshPromise
          originalRequest.headers.Authorization = `Bearer ${accessToken}`
          return api(originalRequest)
        } catch {
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          window.location.href = '/login'
        }
      }
    }

    return Promise.reject(error)
  }
)

export default api
