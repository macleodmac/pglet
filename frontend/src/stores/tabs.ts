import { create } from 'zustand'
import { formatDialect } from '@avallete/sql-formatter/lite'
import { postgresql } from '@avallete/sql-formatter/languages/postgresql'

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
  /** AI tab state */
  aiTurns?: AiTurn[]
  aiCurrentTurnIndex?: number
  aiPrompt?: string
  aiError?: string | null
}

interface TabState {
  tabs: Tab[]
  activeTabId: string
  addQueryTab: () => void
  addTableTab: (tableName: string, preview?: boolean) => void
  addFunctionTab: (functionName: string, preview?: boolean) => void
  addAiTab: () => void
  /** Promote a preview tab to permanent */
  pinTab: (id: string) => void
  closeTab: (id: string) => void
  setActiveTab: (id: string) => void
  updateTab: (id: string, updates: Partial<Tab>) => void
  reorderTabs: (fromIndex: number, toIndex: number) => void
  appendAiTurn: (tabId: string, turn: AiTurn) => void
  setAiTurnIndex: (tabId: string, index: number) => void
}

let nextId = 1
function makeId() {
  return `tab-${nextId++}`
}

function newQueryTab(): Tab {
  const id = makeId()
  return {
    id,
    title: `Query ${nextId - 1}`,
    type: 'query',
    sql: '',
  }
}

const initialTab = newQueryTab()

export const useTabStore = create<TabState>((set, get) => ({
  tabs: [initialTab],
  activeTabId: initialTab.id,

  addQueryTab: () => {
    const tab = newQueryTab()
    set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id }))
  },

  addAiTab: () => {
    const id = makeId()
    const tab: Tab = {
      id,
      title: `AI Query ${nextId - 1}`,
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

    const id = makeId()
    const tab: Tab = {
      id,
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

    const id = makeId()
    const tab: Tab = {
      id,
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

  setActiveTab: (id: string) => set({ activeTabId: id }),

  updateTab: (id: string, updates: Partial<Tab>) => {
    set((s) => ({
      tabs: s.tabs.map((t) => (t.id === id ? { ...t, ...updates } : t)),
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
