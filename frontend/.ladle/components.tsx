import type { GlobalProvider } from '@ladle/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useThemeStore } from '../src/stores/theme'
import '../src/index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
  },
})

export const Provider: GlobalProvider = ({ children, globalState }) => {
  const setTheme = useThemeStore((s) => s.setTheme)
  useEffect(() => {
    setTheme(globalState.theme === 'auto' ? 'system' : globalState.theme)
  }, [globalState.theme, setTheme])

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}
