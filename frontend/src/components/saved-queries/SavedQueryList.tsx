import { useState } from 'react'
import { useSavedQueries, useDeleteSavedQuery } from '../../api/queries'
import { useTabStore } from '../../stores/tabs'
import type { SavedQuery } from '../../api/generated'

interface SavedQueryListProps {
  database?: string
  onEdit?: (query: SavedQuery) => void
}

export function SavedQueryList({ database, onEdit }: SavedQueryListProps) {
  const { data: queries, isLoading } = useSavedQueries(database)
  const deleteMutation = useDeleteSavedQuery()
  const addQueryTab = useTabStore((s) => s.addQueryTab)
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const updateTab = useTabStore((s) => s.updateTab)
  const [search, setSearch] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const filtered = queries?.filter((q) => {
    if (!search) return true
    const s = search.toLowerCase()
    return q.title.toLowerCase().includes(s) || q.tags.toLowerCase().includes(s)
  })

  const handleOpen = (query: SavedQuery) => {
    const activeTab = tabs.find((t) => t.id === activeTabId)
    if (activeTab && activeTab.type === 'query' && !activeTab.sql.trim()) {
      updateTab(activeTab.id, { sql: query.sql, title: query.title, savedQueryId: query.id })
    } else {
      addQueryTab()
      // Update the newly created tab with the saved query
      const newTabs = useTabStore.getState().tabs
      const newTab = newTabs[newTabs.length - 1]
      updateTab(newTab.id, { sql: query.sql, title: query.title, savedQueryId: query.id })
    }
  }

  const handleDelete = (id: string) => {
    deleteMutation.mutate(id, {
      onSuccess: () => setConfirmDelete(null),
    })
  }

  return (
    <div className="flex flex-col">
      <div className="px-3 py-2">
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search saved queries..."
          className="w-full rounded border border-gray-300 bg-white px-2 py-1 text-xs focus:border-accent-400 focus:outline-none dark:border-gray-600 dark:bg-surface-800 dark:text-gray-200"
        />
      </div>

      {isLoading && <p className="px-3 text-xs text-gray-400">Loading...</p>}

      {filtered?.length === 0 && (
        <p className="px-3 text-xs text-gray-400">No saved queries</p>
      )}

      {filtered?.map((q) => (
        <div
          key={q.id}
          className="group flex items-start gap-2 border-b border-surface-200/50 px-3 py-2 hover:bg-surface-50 dark:border-surface-800/50 dark:hover:bg-surface-800/50"
        >
          <button
            onClick={() => handleOpen(q)}
            className="flex-1 text-left"
          >
            <div className="text-xs font-medium text-gray-800 dark:text-gray-200">{q.title}</div>
            {q.description && (
              <div className="mt-0.5 text-[10px] text-gray-500 dark:text-gray-400 line-clamp-1">{q.description}</div>
            )}
            {q.tags && (
              <div className="mt-1 flex gap-1">
                {q.tags.split(',').map((tag) => (
                  <span
                    key={tag.trim()}
                    className="rounded bg-accent-50 px-1.5 py-0.5 text-[10px] text-accent-700 dark:bg-accent-900/30 dark:text-accent-300"
                  >
                    {tag.trim()}
                  </span>
                ))}
              </div>
            )}
          </button>
          <div className="flex gap-1 opacity-0 group-hover:opacity-100">
            {onEdit && (
              <button
                onClick={() => onEdit(q)}
                className="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                title="Edit"
              >
                <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
            )}
            {confirmDelete === q.id ? (
              <>
                <button
                  onClick={() => handleDelete(q.id)}
                  className="rounded bg-red-600 px-1.5 py-0.5 text-[10px] text-white"
                >
                  Delete
                </button>
                <button
                  onClick={() => setConfirmDelete(null)}
                  className="rounded border border-gray-300 px-1.5 py-0.5 text-[10px] text-gray-500 dark:border-gray-600"
                >
                  No
                </button>
              </>
            ) : (
              <button
                onClick={() => setConfirmDelete(q.id)}
                className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                title="Delete"
              >
                <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
