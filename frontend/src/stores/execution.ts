import { create } from 'zustand'
import type { QueryResult } from '../api/generated'

/** In-memory execution state of one tab; never persisted. */
export interface TabRun {
  result: QueryResult | null
  /** SQL that produced `result`, which may be a selection of the editor */
  resultSql: string
  /** Result of the write that safe writes is holding for review */
  heldWrite: QueryResult | null
  isRunning: boolean
  /** A begin, commit or rollback is in flight */
  txPending: boolean
  /** Last failed begin/commit/rollback */
  txError: string | null
}

const emptyRun: TabRun = { result: null, resultSql: '', heldWrite: null, isRunning: false, txPending: false, txError: null }

interface ExecutionState {
  runs: Record<string, TabRun>
  patch: (tabId: string, updates: Partial<TabRun>) => void
  remove: (tabId: string) => void
  /** Forget every held write, e.g. after the server rolled all transactions back */
  clearHeldWrites: () => void
}

export const useExecutionStore = create<ExecutionState>((set) => ({
  runs: {},
  patch: (tabId, updates) =>
    set((s) => ({ runs: { ...s.runs, [tabId]: { ...(s.runs[tabId] ?? emptyRun), ...updates } } })),
  remove: (tabId) =>
    set((s) => {
      const { [tabId]: _, ...runs } = s.runs
      return { runs }
    }),
  clearHeldWrites: () =>
    set((s) => ({
      runs: Object.fromEntries(Object.entries(s.runs).map(([id, r]) => [id, { ...r, heldWrite: null }])),
    })),
}))

export function useTabRun(tabId: string): TabRun {
  return useExecutionStore((s) => s.runs[tabId]) ?? emptyRun
}
