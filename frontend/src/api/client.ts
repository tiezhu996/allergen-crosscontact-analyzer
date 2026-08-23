import axios from 'axios'
import { ElMessage } from 'element-plus'

export interface ApiEnvelope<T> { success: boolean; data: T; request_id: string; meta?: { page: number; page_size: number; total: number; total_pages: number }; error?: { code: string; message: string; details?: unknown } }

export const api = axios.create({ baseURL: '/api/v1', timeout: 15000, headers: { 'Content-Type': 'application/json' } })

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('allergen_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  config.headers['X-Request-ID'] = `web-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status
    const message = error.response?.data?.error?.message || (error.code === 'ECONNABORTED' ? '请求超时' : '服务暂时不可用')
    ElMessage.error(message)
    if (status === 401 && !error.config?.url?.endsWith('/auth/login')) window.dispatchEvent(new Event('auth:expired'))
    return Promise.reject(error)
  },
)

export async function unwrap<T>(request: Promise<{ data: ApiEnvelope<T> }>): Promise<T> { return (await request).data.data }
export async function page<T>(request: Promise<{ data: ApiEnvelope<T[]> }>): Promise<{ items: T[]; total: number }> { const response = (await request).data; return { items: response.data, total: response.meta?.total ?? response.data.length } }
