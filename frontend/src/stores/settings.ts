import { create } from 'zustand'

interface SettingsState {
  /** Hold queries that write in a transaction until the user commits or rolls back */
  safeWrites: boolean
  setSafeWrites: (safeWrites: boolean) => void
}

export const useSettingsStore = create<SettingsState>((set) => ({
  safeWrites: localStorage.getItem('pglet-safe-writes') === 'true',
  setSafeWrites: (safeWrites) => {
    localStorage.setItem('pglet-safe-writes', String(safeWrites))
    set({ safeWrites })
  },
}))
