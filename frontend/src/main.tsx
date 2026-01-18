import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import TorrentUI from './App.tsx'
import { ThemeProvider } from './components/ThemeProvider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <TorrentUI />
    </ThemeProvider>
  </StrictMode>,
)
