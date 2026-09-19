import { useEffect, useState } from 'react'
import { api } from './api'
import { DashboardPage } from './pages/DashboardPage'
import { HomePage } from './pages/HomePage'
import { PollPage } from './pages/PollPage'
import { clearSession, loadSession, saveSession } from './storage'
import type { Session } from './types'
import { navigate } from './utils'

function currentPath(): string {
  return window.location.pathname.replace(/\/+$/, '') || '/'
}

export default function App() {
  const [path, setPath] = useState(currentPath)
  const [session, setSession] = useState<Session | null>(loadSession)

  useEffect(() => {
    const updatePath = () => setPath(currentPath())
    window.addEventListener('popstate', updatePath)
    return () => window.removeEventListener('popstate', updatePath)
  }, [])

  useEffect(() => {
    if (!session) return
    let alive = true
    api.me(session.token).catch(() => {
      if (!alive) return
      clearSession()
      setSession(null)
    })
    return () => { alive = false }
  }, [session?.token])

  function authenticate(nextSession: Session) {
    saveSession(nextSession)
    setSession(nextSession)
    navigate('/')
  }

  function signOut() {
    clearSession()
    setSession(null)
    navigate('/')
  }

  const match = path.match(/^\/p\/([0-9a-f-]{36})$/i)
  if (match) return <PollPage slug={match[1]} session={session} onSignOut={signOut} />
  return session ? <DashboardPage session={session} onSignOut={signOut} /> : <HomePage onAuthenticated={authenticate} />
}
