import { AuthCard } from '../components/AuthCard'
import { Brand, LivePill } from '../components/Brand'
import type { Session } from '../types'

type HomePageProps = {
  onAuthenticated: (session: Session) => void
}

export function HomePage({ onAuthenticated }: HomePageProps) {
  return (
    <main className="home-page">
      <header className="marketing-nav">
        <Brand />
        <a href="#how-it-works">How it works</a>
      </header>
      <section className="hero">
        <div className="hero-copy">
          <LivePill label="Decisions, in motion" />
          <h1>Ask once.<br /><em>See the room</em> decide.</h1>
          <p>PulsePoll makes every vote visible the moment it lands. Create a question, share one link, and watch consensus take shape.</p>
          <button className="primary-button hero-cta" onClick={() => document.getElementById('get-started')?.scrollIntoView({ behavior: 'smooth' })}>Create a live poll <span aria-hidden="true">→</span></button>
          <div className="hero-proof">
            <span className="avatar-stack" aria-hidden="true"><i>A</i><i>M</i><i>R</i><i>+</i></span>
            <span>Built for teams, classrooms<br />and every room with a decision.</span>
          </div>
        </div>
        <div className="hero-visual" aria-label="Example live poll result">
          <div className="signal signal-one" /><div className="signal signal-two" />
          <div className="demo-card">
            <div className="demo-card-top"><span className="tiny-logo">P</span><LivePill /></div>
            <p className="demo-question">What should our next team ritual be?</p>
            <div className="demo-choice active"><span>Friday demo day</span><b>58%</b><i><em /></i></div>
            <div className="demo-choice"><span>Walking 1:1s</span><b>29%</b><i><em /></i></div>
            <div className="demo-choice"><span>Skill swap lunch</span><b>13%</b><i><em /></i></div>
            <div className="demo-footer"><span className="pulse-dot" /> 42 people voting live</div>
          </div>
          <span className="float-note note-one">+1 vote</span>
          <span className="float-note note-two">live results</span>
        </div>
      </section>
      <section id="how-it-works" className="how-it-works">
        <div><span>01</span><h2>Create</h2><p>Set a question and the choices that matter.</p></div>
        <div><span>02</span><h2>Share</h2><p>One clean link works for everyone in the room.</p></div>
        <div><span>03</span><h2>Watch</h2><p>Votes and percentages move in real time.</p></div>
      </section>
      <aside id="get-started" className="auth-wrap"><AuthCard onAuthenticated={onAuthenticated} /></aside>
      <footer>PulsePoll · made for clear decisions</footer>
    </main>
  )
}
