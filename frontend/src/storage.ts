import type { Session } from './types'

const sessionKey = 'pulsepoll.session'

export function loadSession(): Session | null {
  try {
    const value = window.localStorage.getItem(sessionKey)
    return value ? (JSON.parse(value) as Session) : null
  } catch {
    return null
  }
}

export function saveSession(session: Session): void {
  window.localStorage.setItem(sessionKey, JSON.stringify(session))
}

export function clearSession(): void {
  window.localStorage.removeItem(sessionKey)
}

export function hasVoted(slug: string): boolean {
  return window.localStorage.getItem(`pulsepoll.voted.${slug}`) === 'true'
}

export function rememberVote(slug: string): void {
  window.localStorage.setItem(`pulsepoll.voted.${slug}`, 'true')
}
