import { create } from 'zustand'
import { formatDialect } from '@avallete/sql-formatter/lite'
import { postgresql } from '@avallete/sql-formatter/languages/postgresql'
import { getSettings, updateSettings } from '../api/client'

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
}

interface TabState {
  tabs: Tab[]
  activeTabId: string
  /** Whether tabs have been loaded from the server */
  initialized: boolean
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

  initFromServer: async () => {
    try {
      const res = await getSettings()
      const settings = res.data as Record<string, string> | undefined
      const raw = settings?.tabs
      if (!raw) {
        set({ initialized: true })
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
        })
      } else {
        set({ initialized: true })
      }
    } catch {
      set({ initialized: true })
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
    set((s) => {
      const tabs = s.tabs.filter((t) => t.id !== id)
      if (tabs.length === 0) {
        const tab = newQueryTab()
        return { tabs: [tab], activeTabId: tab.id }
      }
      const activeTabId =
        s.activeTabId === id
          ? tabs[Math.min(tabs.findIndex((_, i, arr) => arr[i]?.id !== id) || 0, tabs.length - 1)]?.id ?? tabs[0].id
          : s.activeTabId
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

// --- Debounced auto-save to server ---

function debounce<T extends (...args: never[]) => void>(fn: T, ms: number): T {
  let timer: ReturnType<typeof setTimeout>
  return ((...args: Parameters<T>) => {
    clearTimeout(timer)
    timer = setTimeout(() => fn(...args), ms)
  }) as unknown as T
}

/** Strip transient/preview fields before persisting */
function serializeForSave(state: { tabs: Tab[]; activeTabId: string }): string {
  const payload = {
    tabs: state.tabs
      .filter((t) => !t.preview)
      .map(({ aiPrompt, aiError, preview, ...rest }) => rest),
    activeTabId: state.activeTabId,
  }
  return JSON.stringify(payload)
}

const saveToServer = debounce((state: { tabs: Tab[]; activeTabId: string; initialized: boolean }) => {
  if (!state.initialized) return
  updateSettings({ body: { tabs: serializeForSave(state) } })
}, 1000)

useTabStore.subscribe((state) => saveToServer(state))
