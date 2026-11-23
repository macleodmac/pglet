import { create } from 'zustand'
import { formatDialect } from '@avallete/sql-formatter/lite'
import { postgresql } from '@avallete/sql-formatter/languages/postgresql'
import { getTabState, rollbackTransaction, listOpenTransactions, apiFetch } from '../api/client'
import { useExecutionStore } from './execution'
import { onRelease, useWindowStore } from './window'

function formatSql(sql: string): string {
  if (!sql.trim()) return sql
  try {
    return formatDialect(sql, { dialect: postgresql, tabWidth: 2, keywordCase: 'upper' })
  } catch {
    return sql
  }
}

export interface AiTurn {
  prompt: string
  sql: string
  explanation: string
}

export interface Tab {
  id: string
  title: string
  type: 'query' | 'table' | 'function' | 'ai'
  sql: string
  savedQueryId?: string
  tableName?: string
  functionName?: string
  activeView?: 'rows' | 'structure' | 'indexes' | 'constraints'
  /** Preview tabs are replaced when another preview opens (like VS Code italic tabs) */
  preview?: boolean
  /** Set to true when the user has manually renamed this tab */
  userRenamed?: boolean
  /** Set to true after AI has auto-named this tab (prevents running again) */
  aiRenamed?: boolean
  /** AI tab state */
  aiTurns?: AiTurn[]
  aiCurrentTurnIndex?: number
  aiPrompt?: string
  aiError?: string | null
  /** True while the tab holds an open server-side transaction (not persisted) */
  inTransaction?: boolean
}

interface TabState {
  tabs: Tab[]
  activeTabId: string
  /** Whether tabs have been loaded from the server; nothing is saved before */
  initialized: boolean
  /** Saved tabs could not be loaded, so saving is off to avoid overwriting them */
  loadError: string | null
  addQueryTab: () => void
  addTableTab: (tableName: string, preview?: boolean) => void
  addFunctionTab: (functionName: string, preview?: boolean) => void
  addAiTab: () => void
  /** Promote a preview tab to permanent */
  pinTab: (id: string) => void
  closeTab: (id: string) => void
  setActiveTab: (id: string) => void
  updateTab: (id: string, updates: Partial<Tab>) => void
  renameTab: (id: string, title: string) => void
  reorderTabs: (fromIndex: number, toIndex: number) => void
  appendAiTurn: (tabId: string, turn: AiTurn) => void
  setAiTurnIndex: (tabId: string, index: number) => void
  initFromServer: () => Promise<void>
  /** Mark which tabs hold a server-side transaction, as the server reports it */
  syncOpenTransactions: () => Promise<void>
}

let queryCounter = 1
let aiCounter = 1

function newQueryTab(): Tab {
  return {
    id: crypto.randomUUID(),
    title: `Query ${queryCounter++}`,
    type: 'query',
    sql: '',
  }
}

const initialTab = newQueryTab()

export const useTabStore = create<TabState>((set, get) => ({
  tabs: [initialTab],
  activeTabId: initialTab.id,
  initialized: false,
  loadError: null,

  initFromServer: async () => {
    try {
      const res = await getTabState()
      if (res.error) throw new Error((res.error as { error?: string }).error || 'request failed')
      const raw = (res.data as { data?: string })?.data
      if (!raw) {
        set({ initialized: true, loadError: null })
        await get().syncOpenTransactions()
        return
      }
      const saved = JSON.parse(raw) as { tabs: Tab[]; activeTabId: string }
      if (saved?.tabs?.length) {
        // Restore counters so new tabs don't collide with restored names
        for (const t of saved.tabs) {
          if (t.type === 'query') {
            const m = t.title.match(/^Query (\d+)$/)
            if (m) queryCounter = Math.max(queryCounter, parseInt(m[1], 10) + 1)
          } else if (t.type === 'ai') {
            const m = t.title.match(/^AI Query (\d+)$/)
            if (m) aiCounter = Math.max(aiCounter, parseInt(m[1], 10) + 1)
          }
        }
        set({
          tabs: saved.tabs,
          activeTabId: saved.activeTabId && saved.tabs.some((t) => t.id === saved.activeTabId)
            ? saved.activeTabId
            : saved.tabs[0].id,
          initialized: true,
          loadError: null,
        })
      } else {
        set({ initialized: true, loadError: null })
      }
      await get().syncOpenTransactions()
    } catch (err) {
      set({ loadError: `Saved tabs could not be loaded (${(err as Error).message}); changes to tabs won't be saved until you reload.` })
    }
  },

  syncOpenTransactions: async () => {
    const res = await listOpenTransactions()
    if (!res.data) return
    const open = new Set(res.data.tab_ids)
    set((s) => ({ tabs: s.tabs.map((t) => ({ ...t, inTransaction: open.has(t.id) || undefined })) }))
    const exec = useExecutionStore.getState()
    for (const [id, run] of Object.entries(exec.runs)) {
      if (run.heldWrite && !open.has(id)) exec.patch(id, { heldWrite: null })
    }
  },

  addQueryTab: () => {
    const tab = newQueryTab()
    set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id }))
  },

  addAiTab: () => {
    const tab: Tab = {
      id: crypto.randomUUID(),
      title: `AI Query ${aiCounter++}`,
      type: 'ai',
      sql: '',
    }
    set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id }))
  },

  addTableTab: (tableName: string, preview = false) => {
    const state = get()

    // If the exact table is already open, just activate it (and optionally pin)
    const existing = state.tabs.find(
      (t) => t.type === 'table' && t.tableName === tableName,
    )
    if (existing) {
      const updates: Partial<Tab> = {}
      if (!preview && existing.preview) updates.preview = false
      set((s) => ({
        activeTabId: existing.id,
        tabs: Object.keys(updates).length
          ? s.tabs.map((t) => (t.id === existing.id ? { ...t, ...updates } : t))
          : s.tabs,
      }))
      return
    }

    // If opening a preview, replace the existing preview tab (if any)
    if (preview) {
      const existingPreview = state.tabs.find((t) => t.preview)
      if (existingPreview) {
        set((s) => ({
          tabs: s.tabs.map((t) =>
            t.id === existingPreview.id
              ? { ...t, title: tableName, tableName, activeView: 'rows' as const }
              : t,
          ),
          activeTabId: existingPreview.id,
        }))
        return
      }
    }

    const tab: Tab = {
      id: crypto.randomUUID(),
      title: tableName,
      type: 'table',
      sql: '',
      tableName,
      activeView: 'rows',
      preview: preview || undefined,
    }
    set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id }))
  },

  addFunctionTab: (functionName: string, preview = false) => {
    const state = get()

    const existing = state.tabs.find(
      (t) => t.type === 'function' && t.functionName === functionName,
    )
    if (existing) {
      const updates: Partial<Tab> = {}
      if (!preview && existing.preview) updates.preview = false
      set((s) => ({
        activeTabId: existing.id,
        tabs: Object.keys(updates).length
          ? s.tabs.map((t) => (t.id === existing.id ? { ...t, ...updates } : t))
          : s.tabs,
      }))
      return
    }

    if (preview) {
      const existingPreview = state.tabs.find((t) => t.preview)
      if (existingPreview) {
        set((s) => ({
          tabs: s.tabs.map((t) =>
            t.id === existingPreview.id
              ? { ...t, type: 'function' as const, title: functionName, tableName: undefined, functionName }
              : t,
          ),
          activeTabId: existingPreview.id,
        }))
        return
      }
    }

    const tab: Tab = {
      id: crypto.randomUUID(),
      title: functionName,
      type: 'function',
      sql: '',
      functionName,
      preview: preview || undefined,
    }
    set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id }))
  },

  pinTab: (id: string) => {
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === id ? { ...t, preview: undefined } : t,
      ),
    }))
  },

  closeTab: (id: string) => {
    // Always ask the server: its record of the tab's transaction outlives a
    // reload, while inTransaction here might not be up to date.
    rollbackTransaction({ body: { tab_id: id } }).catch(() => {})
    useExecutionStore.getState().remove(id)
    set((s) => {
      const index = s.tabs.findIndex((t) => t.id === id)
      const tabs = s.tabs.filter((t) => t.id !== id)
      if (tabs.length === 0) {
        const tab = newQueryTab()
        return { tabs: [tab], activeTabId: tab.id }
      }
      const activeTabId = s.activeTabId === id ? tabs[Math.min(index, tabs.length - 1)].id : s.activeTabId
      return { tabs, activeTabId }
    })
  },

  setActiveTab: (id: string) =>
    set((s) => {
      // When switching away from a preview tab, close it (VS Code behavior)
      const tabs = s.tabs.filter((t) => !(t.preview && t.id !== id))
      return { tabs, activeTabId: id }
    }),

  updateTab: (id: string, updates: Partial<Tab>) => {
    set((s) => ({
      tabs: s.tabs.map((t) => (t.id === id ? { ...t, ...updates } : t)),
    }))
  },

  renameTab: (id: string, title: string) => {
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === id ? { ...t, title, userRenamed: true } : t,
      ),
    }))
  },

  reorderTabs: (fromIndex: number, toIndex: number) => {
    set((s) => {
      if (fromIndex === toIndex) return s
      const tabs = [...s.tabs]
      const [moved] = tabs.splice(fromIndex, 1)
      tabs.splice(toIndex, 0, moved)
      return { tabs }
    })
  },

  appendAiTurn: (tabId: string, turn: AiTurn) => {
    set((s) => ({
      tabs: s.tabs.map((t) => {
        if (t.id !== tabId) return t
        const turns = [...(t.aiTurns ?? []), turn]
        return { ...t, aiTurns: turns, aiCurrentTurnIndex: turns.length - 1, sql: formatSql(turn.sql) }
      }),
    }))
  },

  setAiTurnIndex: (tabId: string, index: number) => {
    set((s) => ({
      tabs: s.tabs.map((t) => {
        if (t.id !== tabId) return t
        const turns = t.aiTurns ?? []
        const clamped = Math.max(0, Math.min(index, turns.length - 1))
        return { ...t, aiCurrentTurnIndex: clamped, sql: formatSql(turns[clamped]?.sql ?? '') }
      }),
    }))
  },
}))

// --- Debounced auto-save to the server ---

/** Strip transient/preview fields before persisting */
function serializeForSave(state: { tabs: Tab[]; activeTabId: string }): string {
  const payload = {
    tabs: state.tabs
      .filter((t) => !t.preview)
      .map(({ aiPrompt, aiError, preview, inTransaction, ...rest }) => rest),
    activeTabId: state.activeTabId,
  }
  return JSON.stringify(payload)
}

let saveTimer: ReturnType<typeof setTimeout> | undefined
let pending: string | null = null

function flushSave() {
  clearTimeout(saveTimer)
  if (pending === null) return
  const data = pending
  pending = null
  // keepalive lets the request finish while the page unloads.
  apiFetch('/api/tabs', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data }),
    keepalive: true,
  }).catch(() => {})
}

useTabStore.subscribe((state) => {
  if (!state.initialized || state.loadError || useWindowStore.getState().inactive) return
  pending = serializeForSave(state)
  clearTimeout(saveTimer)
  saveTimer = setTimeout(flushSave, 1000)
})

window.addEventListener('pagehide', flushSave)
onRelease(flushSave)
