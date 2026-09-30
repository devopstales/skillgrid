import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/index.css'
import { App } from './app'
import { applyDensity, getDensity } from './lib/density'

// Apply the persisted density preference before first paint (no flash).
applyDensity(getDensity())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
