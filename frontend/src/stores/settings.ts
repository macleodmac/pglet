import { create } from 'zustand'

/** Row caps offered in the results toolbar; 0 means no cap. */
export const ROW_LIMITS = [100, 1000, 10000, 0] as const
const DEFAULT_MAX_ROWS = 1000
const STORAGE_KEY = 'pglet-max-rows'

function loadMaxRows(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const n = Number(raw)
    return raw !== null && (ROW_LIMITS as readonly number[]).includes(n) ? n : DEFAULT_MAX_ROWS
  } catch {
    return DEFAULT_MAX_ROWS
  }
}

interface SettingsState {
  /** Rows kept per query result; 0 keeps every row. */
  maxRows: number
  setMaxRows: (n: number) => void
}

export const useSettingsStore = create<SettingsState>((set) => ({
  maxRows: loadMaxRows(),
  setMaxRows: (maxRows) => {
    try {
      localStorage.setItem(STORAGE_KEY, String(maxRows))
    } catch {
      // Not persisted; the in-memory value still applies.
    }
    set({ maxRows })
  },
}))
