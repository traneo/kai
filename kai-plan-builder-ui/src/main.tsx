import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles/variables.css'
import './styles/base.css'
import './styles/layout.css'
import './styles/chat.css'
import './styles/ui.css'
import './styles/review.css'
import './styles/builder.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
