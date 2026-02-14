import { useHistory, useClearHistory } from '../../api/queries'
import { useTabStore } from '../../stores/tabs'
import { useState } from 'react'

export function HistoryPanel() {
  const { data, isLoading } = useHistory()
  const clearHistory = useClearHistory()
  const addQueryTab = useTabStore((s) => s.addQueryTab)
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const updateTab = useTabStore((s) => s.updateTab)
  const [confirmClear, setConfirmClear] = useState(false)

  const handleOpen = (sql: string) => {
    const activeTab = tabs.find((t) => t.id === activeTabId)
    if (activeTab && activeTab.type === 'query' && !activeTab.sql.trim()) {
      updateTab(activeTab.id, { sql })
    } else {
      addQueryTab()
      const newTabs = useTabStore.getState().tabs
      const newTab = newTabs[newTabs.length - 1]
      updateTab(newTab.id, { sql })
    }
  }

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between px-3 py-2">
        <span className="text-xs text-gray-500 dark:text-gray-400">
          {data ? `${data.total} entries` : ''}
        </span>
        {confirmClear ? (
          <div className="flex gap-1">
            <button
              type="button"
              onClick={() => {
                clearHistory.mutate(undefined, { onSuccess: () => setConfirmClear(false) })
              }}
              className="rounded bg-red-600 px-1.5 py-0.5 text-[10px] text-white"
            >
              Clear all
            </button>
            <button
              type="button"
              onClick={() => setConfirmClear(false)}
              className="rounded border border-gray-300 px-1.5 py-0.5 text-[10px] text-gray-500 dark:border-gray-600"
            >
              No
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setConfirmClear(true)}
            className="text-[10px] text-gray-400 hover:text-red-500 dark:hover:text-red-400"
          >
            Clear
          </button>
        )}
      </div>

      {isLoading && <p className="px-3 text-xs text-gray-400">Loading...</p>}

      {data?.entries.length === 0 && (
        <p className="px-3 text-xs text-gray-400">No history</p>
      )}

      {data?.entries.map((entry) => (
        <button
          type="button"
          key={entry.id}
          onClick={() => handleOpen(entry.sql)}
          className="border-b border-surface-200/50 px-3 py-1.5 text-left hover:bg-surface-50 dark:border-surface-800/50 dark:hover:bg-surface-800/50"
        >
          <pre className="font-mono truncate text-xs text-gray-700 dark:text-gray-300">
            {entry.sql.slice(0, 100)}
          </pre>
          <div className="mt-0.5 flex gap-2 font-mono text-[10px] text-gray-400">
            <span>{entry.duration_ms}ms</span>
            <span>{entry.row_count} rows</span>
            {entry.error && <span className="text-red-400">error</span>}
            <span className="ml-auto">{new Date(entry.executed_at).toLocaleTimeString()}</span>
          </div>
        </button>
      ))}
    </div>
  )
}
