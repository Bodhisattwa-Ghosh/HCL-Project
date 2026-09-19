import type { AuthPayload, CreatePollPayload, Poll, Session, User } from './types'

const configuredOrigin = import.meta.env.VITE_API_URL?.replace(/\/$/, '') ?? ''
const apiBase = `${configuredOrigin}/api`

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

type RequestOptions = RequestInit & { token?: string }

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { token, headers, ...init } = options
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
  })

  const body = (await response.json().catch(() => ({}))) as { error?: string }
  if (!response.ok) {
    throw new ApiError(body.error ?? 'Something went wrong. Please try again.', response.status)
  }
  return body as T
}

export const api = {
  signUp: (payload: Required<AuthPayload>) =>
    request<Session>('/auth/signup', { method: 'POST', body: JSON.stringify(payload) }),
  login: (payload: Omit<AuthPayload, 'name'>) =>
    request<Session>('/auth/login', { method: 'POST', body: JSON.stringify(payload) }),
  me: (token: string) => request<{ user: User }>('/auth/me', { token }),
  polls: (token: string) => request<{ polls: Poll[] }>('/polls', { token }),
  createPoll: (token: string, payload: CreatePollPayload) =>
    request<{ poll: Poll }>('/polls', { method: 'POST', token, body: JSON.stringify(payload) }),
  poll: (slug: string, token?: string) => request<{ poll: Poll }>(`/polls/${encodeURIComponent(slug)}`, { token }),
  vote: (slug: string, optionId: string) =>
    request<{ poll: Poll }>(`/polls/${encodeURIComponent(slug)}/votes`, {
      method: 'POST',
      body: JSON.stringify({ optionId }),
    }),
  close: (slug: string, token: string) =>
    request<{ poll: Poll }>(`/polls/${encodeURIComponent(slug)}/close`, { method: 'POST', token }),
  eventsUrl: (slug: string) => `${apiBase}/polls/${encodeURIComponent(slug)}/events`,
}
