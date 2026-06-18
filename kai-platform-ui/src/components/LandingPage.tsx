import type { Page } from '../types'
import { NavBar } from './NavBar'

interface Props {
  onEnter: () => void
  onNavigate: (page: Page) => void
}

const features = [
  {
    title: 'Agent Teams',
    desc: 'Define a sequence of AI agents that each own a task — scaffolding, implementation, testing, documentation.',
  },
  {
    title: 'Prompt-Driven',
    desc: 'Write natural language prompts instead of boilerplate. Each agent receives a mission and delivers production-ready code.',
  },
  {
    title: 'Quality Built-In',
    desc: 'Automated validation gates run after every agent task — linters, type checks, tests, and diff review.',
  },
  {
    title: 'Human Oversight',
    desc: 'Optional approval gates let you review agent output before it is committed. Every action is audited.',
  },
  {
    title: 'Live Telemetry',
    desc: 'Watch agents work in real time. Stream stdout, stderr, and status updates via SSE with zero polling.',
  },
  {
    title: 'Multi-Model',
    desc: 'Route each task to the right model — GPT-4o for reasoning, Claude for analysis, GPT-4o-mini for simple tasks.',
  },
]

export function LandingPage({ onEnter, onNavigate }: Props) {
  return (
    <div className="landing">
      <header className="header" style={{ background: 'transparent', borderBottom: '1px solid var(--border)' }}>
        <div className="header-brand">
          <div className="header-logo">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2" />
              <line x1="12" y1="22" x2="12" y2="15.5" />
              <polyline points="22 8.5 12 15.5 2 8.5" />
            </svg>
          </div>
          <h1><span>kai</span> Platform</h1>
        </div>
        <NavBar current="landing" onNavigate={onNavigate} />
      </header>

      <main className="landing-main">
        <section className="landing-hero">
          <h1 className="landing-title">
            AI Agents <span className="landing-title-accent">that Build Software</span>
          </h1>
          <p className="landing-subtitle">
            Describe what you want built. Kai orchestrates specialized AI agents to plan,
            implement, test, and deliver — from a single prompt to a production pull request.
          </p>
          <div className="landing-cta">
            <button className="landing-btn landing-btn-primary" onClick={onEnter}>
              Enter Dashboard
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="5" y1="12" x2="19" y2="12" /><polyline points="12 5 19 12 12 19" />
              </svg>
            </button>
          </div>
        </section>

        <section className="landing-features">
          <h2 className="landing-section-title">How Kai works</h2>
          <div className="landing-grid">
            {features.map(f => (
              <div key={f.title} className="landing-card">
                <h3 className="landing-card-title">{f.title}</h3>
                <p className="landing-card-desc">{f.desc}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="landing-showcase">
          <h2 className="landing-section-title">Intelligent agent orchestration</h2>
          <div className="showcase-steps">
            <div className="showcase-step">
              <span className="showcase-step-number">01</span>
              <div className="showcase-step-content">
                <h3 className="showcase-step-title">Plan</h3>
                <p className="showcase-step-desc">Architect analyzes requirements, designs system architecture, selects the optimal tech stack.</p>
              </div>
            </div>
            <div className="showcase-step">
              <span className="showcase-step-number">02</span>
              <div className="showcase-step-content">
                <h3 className="showcase-step-title">Build</h3>
                <p className="showcase-step-desc">Specialized agents implement database schemas, API endpoints, and frontend components in parallel.</p>
              </div>
            </div>
            <div className="showcase-step">
              <span className="showcase-step-number">03</span>
              <div className="showcase-step-content">
                <h3 className="showcase-step-title">Validate</h3>
                <p className="showcase-step-desc">Automated quality gates run tests, security scans, and compliance checks on every deliverable.</p>
              </div>
            </div>
            <div className="showcase-step">
              <span className="showcase-step-number">04</span>
              <div className="showcase-step-content">
                <h3 className="showcase-step-title">Deliver</h3>
                <p className="showcase-step-desc">Final review triggers PR creation, deployment summary, and rollback plan generation.</p>
              </div>
            </div>
          </div>
        </section>

        <section className="landing-cta-section">
          <h2 className="landing-section-title">Ready to build with AI?</h2>
          <p className="landing-cta-subtitle">
            Define your first mission in YAML and let Kai assemble the team.
          </p>
          <button className="landing-btn landing-btn-primary" onClick={onEnter}>
            Enter Dashboard
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="5" y1="12" x2="19" y2="12" /><polyline points="12 5 19 12 12 19" />
            </svg>
          </button>
        </section>
      </main>

      <footer className="landing-footer">
        <span>Kai</span>
        <span className="landing-footer-dot">·</span>
        <span>Open source</span>
        <span className="landing-footer-dot">·</span>
        <span>MIT License</span>
      </footer>
    </div>
  )
}
