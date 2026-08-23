import { api, page, unwrap } from './client'
import type { AllergenProfile, AuditEvent, ContactEdge, MatrixResult, ProcessRoute, User, VersionDiff } from '@/types/domain'
import type { AssessmentRun, AssessmentStatus } from '@/types/assessment'

export const authApi = {
  login: (username: string, password: string) => unwrap<{ token: string; expires_at: string; user: User }>(api.post('/auth/login', { username, password })),
  me: () => unwrap<User>(api.get('/auth/me')),
}

export const profileApi = {
  list: (params: Record<string, unknown> = {}) => page<AllergenProfile>(api.get('/profiles', { params })),
  detail: (id: number) => unwrap<{ profile: AllergenProfile; used_by_routes: Array<{ route_id: number; route_code: string; product_name: string; route_version: number }> }>(api.get(`/profiles/${id}`)),
  create: (payload: Record<string, unknown>) => unwrap<AllergenProfile>(api.post('/profiles', payload)),
  update: (id: number, payload: Record<string, unknown>) => unwrap<AllergenProfile>(api.put(`/profiles/${id}`, payload)),
}

export const routeApi = {
  list: (params: Record<string, unknown> = {}) => page<ProcessRoute>(api.get('/routes', { params })),
  detail: (id: number) => unwrap<ProcessRoute>(api.get(`/routes/${id}`)),
  create: (payload: Record<string, unknown>) => unwrap<ProcessRoute>(api.post('/routes', payload)),
  update: (id: number, payload: Record<string, unknown>) => unwrap<ProcessRoute>(api.put(`/routes/${id}`, payload)),
}

export const edgeApi = {
  list: (params: Record<string, unknown> = {}) => page<ContactEdge>(api.get('/contact-edges', { params })),
  create: (payload: Record<string, unknown>) => unwrap<ContactEdge>(api.post('/contact-edges', payload)),
  update: (id: number, payload: Record<string, unknown>) => unwrap<ContactEdge>(api.put(`/contact-edges/${id}`, payload)),
}

export const assessmentApi = {
  matrix: (routeId: number) => unwrap<MatrixResult>(api.post('/matrix/compute', { route_id: routeId })),
  list: (params: { route_id?: number; status?: AssessmentStatus } = {}) => page<AssessmentRun>(api.get('/assessments', { params })),
  get: (id: number) => unwrap<AssessmentRun>(api.get(`/assessments/${id}`)),
  create: (routeId: number) => unwrap<AssessmentRun>(api.post('/assessments', { route_id: routeId })),
  run: (id: number) => unwrap<AssessmentRun>(api.post(`/assessments/${id}/run`)),
  review: (id: number, decision: 'accepted' | 'rejected', reason: string) => unwrap<AssessmentRun>(api.post(`/assessments/${id}/review`, { decision, reason })),
}

export const auditApi = {
  list: (params: Record<string, unknown> = {}) => page<AuditEvent>(api.get('/audit', { params })),
  version: (entityType: string, id: number, version: number) => unwrap<VersionDiff>(api.get(`/versions/${entityType}/${id}`, { params: { version } })),
}
