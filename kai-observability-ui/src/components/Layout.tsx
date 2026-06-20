import type { ReactNode } from 'react'
import { NavBar } from './NavBar'
import type { Page } from '../types'

interface Props {
  current: Page
  onNavigate: (page: Page) => void
  children: ReactNode
  title?: string
}

export function Layout({ current, onNavigate, children, title }: Props) {
  return (
    <div className="app-layout">
      <NavBar current={current} onNavigate={onNavigate} />
      <main className="main-content">
        {title && <h1 className="page-title">{title}</h1>}
        {children}
      </main>
    </div>
  )
}
