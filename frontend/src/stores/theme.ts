import { create } from 'zustand'

type Theme = 'light' | 'dark' | 'system'

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
}

function applyTheme(theme: Theme) {
  const root = document.documentElement
  if (theme === 'system') {
    root.classList.remove('light', 'dark')
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      root.classList.add('dark')
    }
  } else {
    root.classList.remove('light', 'dark')
    root.classList.add(theme)
  }
}

// Initialize from localStorage or system
const stored = localStorage.getItem('pglet-theme') as Theme | null
const initial: Theme = stored || 'system'
applyTheme(initial)

export const useThemeStore = create<ThemeState>((set) => ({
  theme: initial,
  setTheme: (theme) => {
    localStorage.setItem('pglet-theme', theme)
    applyTheme(theme)
    set({ theme })
  },
}))
