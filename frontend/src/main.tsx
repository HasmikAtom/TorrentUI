import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import TorrentUI from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <TorrentUI />
  </StrictMode>,
)
