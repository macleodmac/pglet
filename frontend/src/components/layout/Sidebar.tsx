import { useState, useMemo } from 'react'
import { useObjects, useSchemas } from '../../api/queries'
import { useTabStore } from '../../stores/tabs'
import { useConnectionStore } from '../../stores/connection'
import type { SchemaObject } from '../../api/generated'
import { SavedQueryList } from '../saved-queries/SavedQueryList'
import { SaveQueryDialog } from '../saved-queries/SaveQueryDialog'
import { HistoryPanel } from '../history/HistoryPanel'
import type { SavedQuery } from '../../api/generated'

const typeOrder: Record<string, number> = {
  tables: 0,
  views: 1,
  materialized_views: 2,
  functions: 3,
  sequences: 4,
  types: 5,
}

const typeLabels: Record<string, string> = {
  tables: 'Tables',
  views: 'Views',
  materialized_views: 'Materialized Views',
  functions: 'Functions',
  sequences: 'Sequences',
  types: 'Types',
}

type SidebarTab = 'objects' | 'saved' | 'history'

interface SidebarProps {
  activeSection?: SidebarTab
  onSectionChange?: (section: SidebarTab) => void
}

export function Sidebar({ activeSection: controlledActiveSection, onSectionChange }: SidebarProps = {}) {
  const { data: objects, isLoading, refetch } = useObjects()
  const { data: schemas } = useSchemas()
  const addTableTab = useTabStore((s) => s.addTableTab)
  const addFunctionTab = useTabStore((s) => s.addFunctionTab)
  const pinTab = useTabStore((s) => s.pinTab)
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const database = useConnectionStore((s) => s.info?.database ?? '')
  const [selectedSchema, setSelectedSchema] = useState('public')
  const [search, setSearch] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [internalActiveSection, setInternalActiveSection] = useState<SidebarTab>('objects')
  const [editingQuery, setEditingQuery] = useState<SavedQuery | null>(null)

  const activeSection = controlledActiveSection ?? internalActiveSection
  const setActiveSection = (section: SidebarTab) => {
    if (onSectionChange) {
      onSectionChange(section)
    } else {
      setInternalActiveSection(section)
    }
  }

  const toggle = (key: string) =>
    setCollapsed((s) => ({ ...s, [key]: !s[key] }))

  // Get the sorted and filtered groups for the selected schema
  const filteredGroups = useMemo(() => {
    if (!objects || !objects[selectedSchema]) return []
    const groups = objects[selectedSchema]
    const lowerSearch = search.toLowerCase()

    return Object.entries(groups)
      .sort(([a], [b]) => (typeOrder[a] ?? 99) - (typeOrder[b] ?? 99))
      .map(([type, items]) => {
        const objs = items as SchemaObject[]
        const filtered = lowerSearch
          ? objs.filter((o) => o.name.toLowerCase().includes(lowerSearch))
          : objs
        return { type, items: filtered }
      })
      .filter((g) => g.items.length > 0)
  }, [objects, selectedSchema, search])

  const sectionTabs: { key: SidebarTab; label: string }[] = [
    { key: 'objects', label: 'Objects' },
    { key: 'saved', label: 'Saved' },
    { key: 'history', label: 'History' },
  ]

  return (
    <aside className="flex h-full flex-col overflow-hidden border-r border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-900">
      <div className="flex h-9 items-end border-b border-surface-200 dark:border-surface-800">
        <div className="flex flex-1 items-end">
          {sectionTabs.map((st) => (
            <button
              type="button"
              key={st.key}
              onClick={() => setActiveSection(st.key)}
              className={`flex h-8 flex-1 items-center justify-center text-[10px] font-semibold uppercase tracking-wider ${
                activeSection === st.key
                  ? 'border-b-2 border-accent-500 text-accent-600 dark:text-accent-400'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
              }`}
            >
              {st.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto text-sm">
        {activeSection === 'objects' && (
          <>
            {/* Schema selector + refresh */}
            <div className="flex items-center gap-1 border-b border-surface-200 px-2 py-1.5 dark:border-surface-800">
              <select
                value={selectedSchema}
                onChange={(e) => setSelectedSchema(e.target.value)}
                className="min-w-0 flex-1 rounded border border-surface-300 bg-white px-1.5 py-0.5 text-xs text-gray-700 dark:border-surface-700 dark:bg-surface-800 dark:text-gray-300"
              >
                {schemas?.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
              <button
                type="button"
                onClick={() => refetch()}
                className="rounded p-0.5 text-gray-400 hover:bg-surface-200 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                title="Refresh"
              >
                <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>
            </div>

            {/* Search */}
            <div className="border-b border-surface-200 px-2 py-1.5 dark:border-surface-800">
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Filter objects..."
                className="w-full rounded border border-surface-300 bg-white px-2 py-1 text-xs text-gray-700 placeholder:text-gray-400 dark:border-surface-700 dark:bg-surface-800 dark:text-gray-300 dark:placeholder:text-gray-500"
              />
            </div>

            {isLoading && (
              <p className="p-3 text-xs text-gray-400">Loading...</p>
            )}
            {filteredGroups.map(({ type, items }) => {
              const key = `${selectedSchema}.${type}`
              return (
                <div key={key}>
                  <button
                    type="button"
                    onClick={() => toggle(key)}
                    className="flex w-full items-center gap-1 px-3 py-1 text-left text-xs text-gray-500 hover:bg-surface-100 dark:text-gray-400 dark:hover:bg-surface-800"
                  >
                    <span className="text-[10px]">{collapsed[key] ? '▶' : '▼'}</span>
                    {typeLabels[type] || type} ({items.length})
                  </button>
                  {!collapsed[key] &&
                    items.map((obj) => {
                      const isTableLike = type === 'tables' || type === 'views' || type === 'materialized_views'
                      const isFunction = type === 'functions'
                      const isBrowsable = isTableLike || isFunction
                      const qualifiedName = selectedSchema === 'public' ? obj.name : `${selectedSchema}.${obj.name}`
                      const isActive = isBrowsable && tabs.some(
                        (t) => t.id === activeTabId && (
                          (isTableLike && t.type === 'table' && t.tableName === qualifiedName) ||
                          (isFunction && t.type === 'function' && t.functionName === qualifiedName)
                        ),
                      )
                      return (
                        <button
                          type="button"
                          key={obj.name}
                          onClick={() => {
                            if (isFunction) addFunctionTab(qualifiedName, true)
                            else if (isTableLike) addTableTab(qualifiedName, true)
                          }}
                          onDoubleClick={() => {
                            if (isFunction) {
                              addFunctionTab(qualifiedName)
                              const previewTab = tabs.find(
                                (t) => t.type === 'function' && t.functionName === qualifiedName && t.preview,
                              )
                              if (previewTab) pinTab(previewTab.id)
                            } else if (isTableLike) {
                              addTableTab(qualifiedName)
                              const previewTab = tabs.find(
                                (t) => t.type === 'table' && t.tableName === qualifiedName && t.preview,
                              )
                              if (previewTab) pinTab(previewTab.id)
                            }
                          }}
                          className={`block w-full truncate px-6 py-0.5 text-left text-xs font-mono ${
                            isActive
                              ? 'bg-accent-50 text-accent-700 dark:bg-accent-900/20 dark:text-accent-300'
                              : 'text-gray-700 hover:bg-accent-50 hover:text-accent-700 dark:text-gray-300 dark:hover:bg-accent-900/20 dark:hover:text-accent-300'
                          }`}
                          title={obj.comment || obj.name}
                        >
                          {obj.name}
                        </button>
                      )
                    })}
                </div>
              )
            })}
            {!isLoading && filteredGroups.length === 0 && search && (
              <p className="p-3 text-xs text-gray-400">No matches</p>
            )}
          </>
        )}

        {activeSection === 'saved' && (
          <SavedQueryList database={database} onEdit={(q) => setEditingQuery(q)} />
        )}

        {activeSection === 'history' && <HistoryPanel />}
      </div>

      {editingQuery && (
        <SaveQueryDialog
          sql={editingQuery.sql}
          existingQuery={editingQuery}
          database={database}
          onClose={() => setEditingQuery(null)}
        />
      )}

    </aside>
  )
}
