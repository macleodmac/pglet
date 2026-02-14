import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import { queryClient } from './queryClient'
import './index.css'

const app = (
  <QueryClientProvider client={queryClient}>
    <App />
  </QueryClientProvider>
)

createRoot(document.getElementById('root')!).render(
  import.meta.env.DEV ? <StrictMode>{app}</StrictMode> : app,
)
