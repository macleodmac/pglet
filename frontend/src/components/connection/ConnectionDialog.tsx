import { useCallback, useEffect, useRef, useState } from 'react'
import { useConnect } from '../../api/queries'
import { useConnectionStore } from '../../stores/connection'

interface SavedConnection {
  name: string
  url: string
}

const STORAGE_KEY = 'pglet-connections'

function loadSaved(): SavedConnection[] {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]')
  } catch {
    return []
  }
}

function persistSaved(connections: SavedConnection[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(connections))
}

/** Derive a display name from a postgres URL */
function nameFromUrl(raw: string): string {
  try {
    const u = new URL(raw)
    const db = u.pathname.replace(/^\//, '') || 'postgres'
    const host = u.hostname || 'localhost'
    const user = u.username || 'postgres'
    return `${user}@${host}/${db}`
  } catch {
    return raw.slice(0, 40)
  }
}

export function ConnectionDialog() {
  const [url, setUrl] = useState('postgres://localhost:5432/postgres')
  const [saved, setSaved] = useState<SavedConnection[]>(loadSaved)
  const [editingName, setEditingName] = useState<string | null>(null)
  const [nameInput, setNameInput] = useState('')
  const connect = useConnect()
  const setConnected = useConnectionStore((s) => s.setConnected)

  const urlInputRef = useRef<HTMLInputElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)

  // Focus URL input on mount
  useEffect(() => {
    urlInputRef.current?.focus()
  }, [])

  // Focus name input when editing starts
  useEffect(() => {
    if (editingName) nameInputRef.current?.focus()
  }, [editingName])

  const handleConnect = useCallback(
    (connectUrl?: string) => {
      const target = connectUrl || url
      if (!target.trim()) return
      connect.mutate(target, {
        onSuccess: (info) => {
          setConnected({
            host: info.host,
            port: info.port,
            user: info.user,
            database: info.database,
            version: info.version,
          })
        },
      })
    },
    [url, connect, setConnected],
  )

  const handleSave = () => {
    if (!url.trim()) return
    const alreadySaved = saved.some((c) => c.url === url)
    if (alreadySaved) return
    setEditingName(url)
    setNameInput(nameFromUrl(url))
  }

  const confirmSave = () => {
    if (!editingName) return
    const name = nameInput.trim() || nameFromUrl(editingName)
    const next = [...saved, { name, url: editingName }]
    setSaved(next)
    persistSaved(next)
    setEditingName(null)
    setNameInput('')
  }

  const handleRemove = (removeUrl: string) => {
    const next = saved.filter((c) => c.url !== removeUrl)
    setSaved(next)
    persistSaved(next)
  }

  const alreadySaved = saved.some((c) => c.url === url)

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-50 dark:bg-surface-950">
      <div className="w-full max-w-md animate-slide-up rounded-xl border border-surface-200 bg-white p-6 shadow-lg shadow-black/5 dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/30">
        <div className="mb-1 flex items-center gap-2">
          <img src="/icon.png" alt="" className="h-16 w-16" aria-hidden="true" />
          <h1 className="font-mono text-xl font-semibold text-gray-900 dark:text-gray-100">pglet</h1>
        </div>
        <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">
          Connect to a PostgreSQL database
        </p>

        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Connection URL
        </label>
        <div className="mb-4 flex gap-2">
          <input
            ref={urlInputRef}
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleConnect()}
            placeholder="postgres://user:pass@host:port/db"
            className="block w-full rounded-md border border-gray-300 bg-white px-3 py-2 font-mono text-sm shadow-sm focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-700 dark:bg-surface-800 dark:text-gray-100"
          />
          <button
            type="button"
            onClick={handleSave}
            disabled={!url.trim() || alreadySaved}
            className="shrink-0 rounded-md border border-surface-300 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-surface-100 disabled:opacity-30 dark:border-surface-700 dark:text-gray-400 dark:hover:bg-surface-800"
            title={alreadySaved ? 'Already saved' : 'Save connection'}
          >
            {alreadySaved ? 'Saved' : 'Save'}
          </button>
        </div>

        {/* Inline name editor */}
        {editingName && (
          <div className="mb-4 flex gap-2">
            <input
              ref={nameInputRef}
              type="text"
              value={nameInput}
              onChange={(e) => setNameInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && confirmSave()}
              placeholder="Connection name"
              className="block w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-700 dark:bg-surface-800 dark:text-gray-100"
            />
            <button
              type="button"
              onClick={confirmSave}
              className="shrink-0 rounded-md bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600"
            >
              OK
            </button>
            <button
              type="button"
              onClick={() => setEditingName(null)}
              className="shrink-0 rounded-md border border-surface-300 px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-400 dark:hover:bg-surface-800"
            >
              Cancel
            </button>
          </div>
        )}

        {connect.error && (
          <p className="mb-4 text-sm text-danger">
            {connect.error.message}
          </p>
        )}

        <button
          type="button"
          onClick={() => handleConnect()}
          disabled={connect.isPending || !url.trim()}
          className="mb-5 w-full rounded-md bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 focus:outline-none focus:ring-2 focus:ring-accent-400 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-surface-900"
        >
          {connect.isPending ? 'Connecting...' : 'Connect'}
        </button>

        {/* Saved connections */}
        {saved.length > 0 && (
          <div>
            <p className="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">
              Saved connections
            </p>
            <div className="space-y-1">
              {saved.map((c) => (
                <div
                  key={c.url}
                  className="group flex items-center gap-2 rounded-md border border-surface-200 px-3 py-2 hover:border-accent-300 hover:bg-accent-50/50 dark:border-surface-800 dark:hover:border-accent-700 dark:hover:bg-accent-900/20"
                >
                  <button
                    type="button"
                    onClick={() => {
                      setUrl(c.url)
                      handleConnect(c.url)
                    }}
                    className="flex min-w-0 flex-1 flex-col text-left"
                  >
                    <span className="text-sm font-medium text-gray-800 dark:text-gray-200">
                      {c.name}
                    </span>
                    <span className="truncate font-mono text-xs text-gray-400 dark:text-gray-500">
                      {c.url}
                    </span>
                  </button>
                  <button
                    type="button"
                    onClick={() => handleRemove(c.url)}
                    className="shrink-0 rounded p-1 text-gray-300 opacity-0 hover:text-danger group-hover:opacity-100 dark:text-gray-600 dark:hover:text-danger"
                    aria-label="Remove connection"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="size-3.5" aria-hidden="true">
                      <path d="M5.28 4.22a.75.75 0 0 0-1.06 1.06L6.94 8l-2.72 2.72a.75.75 0 1 0 1.06 1.06L8 9.06l2.72 2.72a.75.75 0 1 0 1.06-1.06L9.06 8l2.72-2.72a.75.75 0 0 0-1.06-1.06L8 6.94 5.28 4.22Z" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
