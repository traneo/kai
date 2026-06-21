import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles/variables.css'
import './styles/base.css'
import './styles/layout.css'
import './styles/ui.css'
import './styles/refresh-bar.css'
import './styles/logging.css'
import './styles/live-stream.css'
import './styles/cards.css'
import './styles/dashboard.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>
)
