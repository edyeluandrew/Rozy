const API = import.meta.env.VITE_API_URL ?? 'http://localhost:8080/v1'

export function getToken() {
  return localStorage.getItem('rozy_admin_token') ?? ''
}

export function setToken(token: string) {
  localStorage.setItem('rozy_admin_token', token)
}

export function clearToken() {
  localStorage.removeItem('rozy_admin_token')
}

async function api<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    ...(opts.headers as Record<string, string>),
  }
  if (opts.body && !(opts.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(`${API}${path}`, { ...opts, headers })
  const text = await res.text()
  const data = text ? JSON.parse(text) : {}
  if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`)
  return data as T
}

export const authApi = {
  requestOtp: (phone: string) =>
    api('/auth/otp/request', { method: 'POST', body: JSON.stringify({ phone }) }),
  verifyOtp: (phone: string, code: string) =>
    api<{ token: string; user: { role: string } }>('/auth/otp/verify', {
      method: 'POST',
      body: JSON.stringify({ phone, code, role: 'admin' }),
    }),
}

export type QueueItem = {
  submission_id: string
  operator_id: string
  legal_name: string
  phone: string
  ride_type: string
  plate: string
  status: string
  submitted_at: string
}

export type SubmissionDetail = QueueItem & {
  permit_number: string
  permit_expiry?: string
  nin_last4?: string
  documents: { id: string; doc_type: string; storage_key: string; mime_type: string }[]
}

export type ActiveTrip = {
  id: string
  status: string
  ride_type: string
  estimated_fare?: number
  passenger_phone: string
  driver_name?: string
  driver_plate?: string
  pickup_lat: number
  pickup_lng: number
  driver_lat?: number
  driver_lng?: number
  created_at: string
  assigned_at?: string
}

export type OperatorItem = {
  id: string
  phone: string
  name: string
  ride_type: string
  status: string
  plate?: string
  wallet_balance: number
  wallet_min_balance: number
  verified: boolean
}

export const adminApi = {
  stats: () => api<{ pending_verifications: number; active_trips: number }>('/admin/stats'),
  queue: () => api<{ items: QueueItem[] }>('/admin/verification/queue'),
  detail: (id: string) => api<SubmissionDetail>(`/admin/verification/${id}`),
  approve: (id: string) => api(`/admin/verification/${id}/approve`, { method: 'POST' }),
  reject: (id: string, reason: string) =>
    api(`/admin/verification/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  activeTrips: () => api<{ trips: ActiveTrip[] }>('/admin/trips/active'),
  operators: (status?: string) =>
    api<{ operators: OperatorItem[] }>(
      status ? `/admin/operators?status=${encodeURIComponent(status)}` : '/admin/operators',
    ),
  fileUrl: (storageKey: string) =>
    `${API}/admin/files/${storageKey}?token=${encodeURIComponent(getToken())}`,
}
