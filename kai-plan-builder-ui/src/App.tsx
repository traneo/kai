import { useState } from 'react'
import type { Page } from './types'
import { NavBar } from './components/NavBar'
import { PlanBuilder } from './components/PlanBuilder'
import { PipelineReview } from './components/PipelineReview'

export default function App() {
  const [page, setPage] = useState<Page>('builder')
  const [yaml, setYaml] = useState('')

  function handleContinue(_cid: string, _currentSpec: string, generatedYaml: string) {
    setYaml(generatedYaml)
    setPage('review')
  }

  function handleGoBack() {
    setPage('builder')
  }

  function handleCancel() {
    setYaml('')
    setPage('builder')
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-brand">
          <div className="header-logo">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2" />
              <line x1="12" y1="22" x2="12" y2="15.5" />
              <polyline points="22 8.5 12 15.5 2 8.5" />
            </svg>
          </div>
          <h1><span>kai</span> Plan Builder</h1>
        </div>
        <NavBar
          current={page}
          onGoBack={page === 'review' ? handleGoBack : undefined}
        />
      </header>

      {page === 'builder' && (
        <PlanBuilder
          onContinue={handleContinue}
        />
      )}

      {page === 'review' && (
        <PipelineReview
          yaml={yaml}
          onGoBack={handleGoBack}
          onCancel={handleCancel}
        />
      )}
    </div>
  )
}
