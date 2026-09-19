import { useState, type FormEvent } from 'react'
import { ApiError, api } from '../api'
import type { Session } from '../types'
import { Toast } from './Toast'

type Mode = 'login' | 'signup'

type AuthCardProps = {
  onAuthenticated: (session: Session) => void
  initialMode?: Mode
}

export function AuthCard({ onAuthenticated, initialMode = 'signup' }: AuthCardProps) {
  const [mode, setMode] = useState<Mode>(initialMode)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      const session = mode === 'signup'
        ? await api.signUp({ name, email, password })
        : await api.login({ email, password })
      onAuthenticated(session)
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not connect to PulsePoll. Try again shortly.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="auth-card" aria-labelledby="auth-heading">
      <div className="eyebrow">For poll creators</div>
      <h2 id="auth-heading">{mode === 'signup' ? 'Make decisions move.' : 'Welcome back.'}</h2>
      <p>{mode === 'signup' ? 'Create a free account to launch and manage your polls.' : 'Sign in to see your live polls.'}</p>
      <form onSubmit={submit} noValidate>
        {mode === 'signup' && (
          <label>
            Your name
            <input value={name} onChange={(event) => setName(event.target.value)} maxLength={60} placeholder="Avery Chen" autoComplete="name" required />
          </label>
        )}
        <label>
          Email address
          <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" placeholder="you@example.com" autoComplete="email" required />
        </label>
        <label>
          Password
          <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" placeholder="10+ characters, letter + number" autoComplete={mode === 'signup' ? 'new-password' : 'current-password'} required />
        </label>
        {error && <Toast message={error} />}
        <button className="primary-button wide" disabled={busy} type="submit">
          {busy ? 'One moment…' : mode === 'signup' ? 'Create my account' : 'Sign in'}
          {!busy && <span aria-hidden="true">→</span>}
        </button>
      </form>
      <button className="text-button auth-switch" type="button" onClick={() => { setMode(mode === 'signup' ? 'login' : 'signup'); setError('') }}>
        {mode === 'signup' ? 'Already have an account? Sign in' : 'New to PulsePoll? Create an account'}
      </button>
    </section>
  )
}
