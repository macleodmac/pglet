import { useRef, useState } from 'react'
import { useTabStore } from '../../stores/tabs'
import { SettingsDialog } from '../ai/SettingsDialog'
import { Icon } from '../ui/Icon'

export function TabBar() {
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const setActiveTab = useTabStore((s) => s.setActiveTab)
  const closeTab = useTabStore((s) => s.closeTab)
  const addQueryTab = useTabStore((s) => s.addQueryTab)
  const addAiTab = useTabStore((s) => s.addAiTab)
  const pinTab = useTabStore((s) => s.pinTab)
  const reorderTabs = useTabStore((s) => s.reorderTabs)
  const [showSettings, setShowSettings] = useState(false)
  const dragIndexRef = useRef<number | null>(null)
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)

  return (
    <>
      <div className="flex h-9 items-end border-b border-surface-200 bg-surface-100 dark:border-surface-800 dark:bg-surface-900">
        <div className="flex min-w-0 flex-1 items-end gap-0 overflow-x-auto">
          {tabs.map((tab, index) => {
            const active = tab.id === activeTabId
            return (
              <button
                type="button"
                key={tab.id}
                draggable
                onDragStart={(e) => {
                  dragIndexRef.current = index
                  e.dataTransfer.effectAllowed = 'move'
                }}
                onDragOver={(e) => {
                  e.preventDefault()
                  e.dataTransfer.dropEffect = 'move'
                  setDragOverIndex(index)
                }}
                onDragLeave={() => setDragOverIndex(null)}
                onDrop={(e) => {
                  e.preventDefault()
                  if (dragIndexRef.current !== null) {
                    reorderTabs(dragIndexRef.current, index)
                  }
                  dragIndexRef.current = null
                  setDragOverIndex(null)
                }}
                onDragEnd={() => {
                  dragIndexRef.current = null
                  setDragOverIndex(null)
                }}
                onClick={() => setActiveTab(tab.id)}
                onDoubleClick={() => {
                  if (tab.preview) pinTab(tab.id)
                }}
                className={`group relative flex h-8 max-w-48 items-center gap-1 border-r border-surface-200 px-3 text-xs dark:border-surface-800 ${
                  active
                    ? 'bg-white text-gray-900 dark:bg-surface-950 dark:text-gray-100'
                    : 'bg-surface-100 text-gray-500 hover:bg-surface-50 dark:bg-surface-900 dark:text-gray-400 dark:hover:bg-surface-800'
                } ${dragOverIndex === index ? 'border-l-2 border-l-accent-500' : ''}`}
              >
                {tab.type === 'table' ? (
                  <Icon name="table" className="h-3 w-3 shrink-0 opacity-50" />
                ) : tab.type === 'function' ? (
                  <Icon name="code" className="h-3 w-3 shrink-0 opacity-50" />
                ) : tab.type === 'ai' ? (
                  <Icon name="wand" className="h-3 w-3 shrink-0 opacity-50" />
                ) : (
                  <Icon name="bolt" className="h-3 w-3 shrink-0 opacity-50" />
                )}
                <span className={`truncate ${tab.preview ? 'italic' : ''}`}>{tab.title}</span>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    closeTab(tab.id)
                  }}
                  className="ml-1 rounded p-0.5 text-gray-400 opacity-0 hover:bg-gray-200 hover:text-gray-600 group-hover:opacity-100 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                  aria-label="Close tab"
                >
                  <Icon name="xmark" className="h-2.5 w-2.5" />
                </button>
                {active && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-accent-500" />
                )}
              </button>
            )
          })}
          <button
            type="button"
            onClick={addQueryTab}
            className="flex h-8 w-8 shrink-0 items-center justify-center text-gray-400 hover:bg-surface-50 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
            title="New query tab"
          >
            <Icon name="bolt" className="h-3 w-3" />
          </button>
          <button
            type="button"
            onClick={addAiTab}
            className="flex h-8 w-8 shrink-0 items-center justify-center text-gray-400 hover:bg-surface-50 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
            title="New AI Query tab"
          >
            <Icon name="wand" className="h-3 w-3" />
          </button>
        </div>
        <button
          type="button"
          onClick={() => setShowSettings(true)}
          className="flex h-8 w-8 shrink-0 items-center justify-center text-gray-400 hover:bg-surface-50 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
          title="Settings"
        >
          <Icon name="gear" className="h-3 w-3" />
        </button>
      </div>
      {showSettings && <SettingsDialog onClose={() => setShowSettings(false)} />}
    </>
  )
}
