import { useEffect, useMemo, useState } from 'react'
import { ApiError, api } from '../api'
import { Brand, LivePill } from '../components/Brand'
import { PollResults } from '../components/PollResults'
import { ShareLink } from '../components/ShareLink'
import { Toast } from '../components/Toast'
import { hasVoted, rememberVote } from '../storage'
import type { Poll, Session } from '../types'
import { formatCount, formatDate, navigate, shortDate } from '../utils'

type PollPageProps = {
  slug: string
  session: Session | null
  onSignOut: () => void
}

export function PollPage({ slug, session, onSignOut }: PollPageProps) {
  const [poll, setPoll] = useState<Poll | null>(null)
  const [selected, setSelected] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [voteError, setVoteError] = useState('')
  const [voting, setVoting] = useState(false)
  const [voted, setVoted] = useState(() => hasVoted(slug))
  const [live, setLive] = useState(false)
  const [closing, setClosing] = useState(false)

  useEffect(() => {
    let alive = true
    setLoading(true)
    setError('')
    setVoted(hasVoted(slug))
    api.poll(slug, session?.token)
      .then(({ poll: next }) => { if (alive) setPoll(next) })
      .catch((caught) => { if (alive) setError(caught instanceof ApiError ? caught.message : 'This poll could not be loaded.') })
      .finally(() => { if (alive) setLoading(false) })
    return () => { alive = false }
  }, [slug, session?.token])

  useEffect(() => {
    const events = new EventSource(api.eventsUrl(slug), { withCredentials: true })
    events.addEventListener('open', () => setLive(true))
    events.addEventListener('results', (event) => {
      try {
        const next = JSON.parse((event as MessageEvent<string>).data) as Poll
        setPoll((current) => current ? { ...next, isOwner: current.isOwner } : next)
        setLive(true)
      } catch {
        // Ignore malformed transport messages and keep the last known result.
      }
    })
    events.addEventListener('error', () => setLive(false))
    return () => events.close()
  }, [slug])

  const voteLabel = useMemo(() => {
    if (poll?.isClosed) return 'Poll closed'
    if (voted) return 'Your vote is in'
    if (voting) return 'Sending your vote…'
    return selected ? 'Cast your vote' : 'Choose an option'
  }, [poll?.isClosed, selected, voted, voting])

  async function castVote() {
    if (!selected || !poll || poll.isClosed || voted) return
    setVoteError('')
    setVoting(true)
    try {
      const { poll: next } = await api.vote(slug, selected)
      setPoll((current) => ({ ...next, isOwner: current?.isOwner ?? false }))
      rememberVote(slug)
      setVoted(true)
    } catch (caught) {
      const message = caught instanceof ApiError ? caught.message : 'Your vote could not be sent. Try again.'
      setVoteError(message)
      if (caught instanceof ApiError && caught.status === 409 && message.includes('already voted')) setVoted(true)
    } finally {
      setVoting(false)
    }
  }

  async function closePoll() {
    if (!poll || !session) return
    setClosing(true)
    setVoteError('')
    try {
      const { poll: next } = await api.close(slug, session.token)
      setPoll(next)
    } catch (caught) {
      setVoteError(caught instanceof ApiError ? caught.message : 'Could not close this poll.')
    } finally {
      setClosing(false)
    }
  }

  return (
    <main className="poll-page">
      <header className="app-nav poll-nav">
        <Brand />
        <div className="nav-actions">
          {session ? <><button className="text-button" onClick={() => navigate('/')}>My polls</button><button className="text-button" onClick={onSignOut}>Sign out</button></> : <button className="secondary-button compact" onClick={() => navigate('/')}>Create a poll</button>}
        </div>
      </header>
      {loading && <div className="loading-card poll-loading"><span className="spinner" /> Opening the live room…</div>}
      {!loading && error && <section className="not-found"><div className="empty-orbit"><span>!</span></div><h1>We couldn’t open this poll.</h1><p>{error}</p><button className="primary-button" onClick={() => navigate('/')}>Go to PulsePoll</button></section>}
      {!loading && poll && (
        <section className="poll-layout">
          <article className="poll-main-card">
            <div className="poll-meta-row"><LivePill label={poll.isClosed ? 'Voting ended' : live ? 'Live now' : 'Reconnecting…'} /><span>{poll.isClosed ? 'Final result' : shortDate(poll.closesAt)}</span></div>
            <h1>{poll.question}</h1>
            {!poll.isClosed && !voted && <p className="poll-instruction">Pick one answer. Results update for everyone as each vote lands.</p>}
            {voted && !poll.isClosed && <div className="voted-notice"><span>✓</span> Your vote is in. Keep watching the room decide.</div>}
            <PollResults poll={poll} selectedId={selected} onSelect={setSelected} disabled={poll.isClosed || voted || voting} showSelection={!voted && !poll.isClosed} />
            <div className="vote-action-row">
              <span><b>{formatCount(poll.totalVotes)}</b> live vote{poll.totalVotes === 1 ? '' : 's'}</span>
              {!poll.isClosed && <button className="primary-button" onClick={castVote} disabled={!selected || voting || voted}>{voteLabel} {!voting && !voted && <span aria-hidden="true">→</span>}</button>}
            </div>
            {voteError && <Toast message={voteError} />}
          </article>
          <aside className="poll-sidebar">
            <ShareLink slug={poll.slug} />
            <div className="sidebar-detail"><span className={live ? 'connection online' : 'connection'} /><div><b>{live ? 'Live connection' : 'Connecting'}</b><p>{live ? 'Results sync automatically.' : 'Trying to reconnect to results.'}</p></div></div>
            <div className="sidebar-detail"><span className="calendar-icon" aria-hidden="true">◷</span><div><b>{poll.isClosed ? 'Poll closed' : 'Timing'}</b><p>{poll.closesAt ? (poll.isClosed ? `Closed ${formatDate(poll.closesAt)}` : `Closes ${formatDate(poll.closesAt)}`) : 'Open until the creator closes it.'}</p></div></div>
            {poll.isOwner && !poll.isClosed && <button className="danger-button" onClick={closePoll} disabled={closing}>{closing ? 'Closing…' : 'Close voting'}</button>}
          </aside>
        </section>
      )}
      <footer className="poll-footer"><span className="pulse-dot" /> Results update live — no refresh required</footer>
    </main>
  )
}
