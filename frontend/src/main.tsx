import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import App from './App'
import { PlayerProvider } from './state/player'
import { queryClient } from './lib/queryClient'
import './styles.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <PlayerProvider>
          <App />
          <Toaster position="top-right" richColors />
        </PlayerProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>
)

