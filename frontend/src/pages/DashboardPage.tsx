import { useEffect, useState } from 'react'
import { ApiError, api } from '../api'
import { Brand, LivePill } from '../components/Brand'
import { CreatePollForm } from '../components/CreatePollForm'
import { ShareLink } from '../components/ShareLink'
import { Toast } from '../components/Toast'
import type { Poll, Session } from '../types'
import { formatCount, navigate, shortDate } from '../utils'

type DashboardPageProps = {
  session: Session
  onSignOut: () => void
}

export function DashboardPage({ session, onSignOut }: DashboardPageProps) {
  const [polls, setPolls] = useState<Poll[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    let alive = true
    api.polls(session.token)
      .then(({ polls: nextPolls }) => { if (alive) setPolls(nextPolls) })
      .catch((caught) => { if (alive) setError(caught instanceof ApiError ? caught.message : 'Could not load your polls.') })
      .finally(() => { if (alive) setLoading(false) })
    return () => { alive = false }
  }, [session.token])

  function onCreated(poll: Poll) {
    setPolls((current) => [poll, ...current])
    setCreating(false)
    navigate(`/p/${poll.slug}`)
  }

  return (
    <main className="app-shell">
      <header className="app-nav">
        <Brand />
        <div className="nav-actions"><span className="account-name">Hi, {session.user.name.split(' ')[0]}</span><button className="text-button" onClick={onSignOut}>Sign out</button></div>
      </header>
      <section className="dashboard-hero">
        <div><LivePill label="Your control room" /><h1>Make the next<br /><em>choice count.</em></h1><p>Create a question, bring in the room, and let the live signal do the talking.</p></div>
        <button className="primary-button hero-action" onClick={() => setCreating(true)}>+ Create a poll</button>
      </section>
      {creating && <CreatePollForm token={session.token} onCreated={onCreated} onCancel={() => setCreating(false)} />}
      <section className="polls-section" aria-labelledby="your-polls-title">
        <div className="section-heading"><div><div className="eyebrow">Workspace</div><h2 id="your-polls-title">Your polls</h2></div>{!creating && polls.length > 0 && <button className="secondary-button compact" onClick={() => setCreating(true)}>+ New poll</button>}</div>
        {error && <Toast message={error} />}
        {loading ? <div className="loading-card"><span className="spinner" /> Loading your live polls…</div> : polls.length === 0 ? <EmptyState onCreate={() => setCreating(true)} /> : <div className="poll-grid">{polls.map((poll) => <PollCard key={poll.id} poll={poll} />)}</div>}
      </section>
    </main>
  )
}

function PollCard({ poll }: { poll: Poll }) {
  return (
    <article className="poll-card">
      <div className="poll-card-top"><span className={poll.isClosed ? 'status-closed' : 'status-live'}>{poll.isClosed ? 'Closed' : <><span />Live</>}</span><span className="poll-date">{shortDate(poll.closesAt)}</span></div>
      <button className="poll-card-question" onClick={() => navigate(`/p/${poll.slug}`)}>{poll.question}</button>
      <div className="poll-card-foot"><span><b>{formatCount(poll.totalVotes)}</b> vote{poll.totalVotes === 1 ? '' : 's'}</span><ShareLink slug={poll.slug} subtle /></div>
    </article>
  )
}

function EmptyState({ onCreate }: { onCreate: () => void }) {
  return <div className="empty-state"><div className="empty-orbit"><span>?</span></div><h3>Your first question is waiting.</h3><p>Start a poll and turn a roomful of opinions into a shared signal.</p><button className="primary-button" onClick={onCreate}>Create your first poll <span aria-hidden="true">→</span></button></div>
}
