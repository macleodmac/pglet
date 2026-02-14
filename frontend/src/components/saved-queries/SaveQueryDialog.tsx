import { useState } from 'react'
import { useCreateSavedQuery, useUpdateSavedQuery } from '../../api/queries'
import type { SavedQuery } from '../../api/generated'

interface SaveQueryDialogProps {
  sql: string
  existingQuery?: SavedQuery
  database?: string
  onClose: () => void
  onSaved?: () => void
}

export function SaveQueryDialog({ sql, existingQuery, database, onClose, onSaved }: SaveQueryDialogProps) {
  const [title, setTitle] = useState(existingQuery?.title ?? '')
  const [description, setDescription] = useState(existingQuery?.description ?? '')
  const [tags, setTags] = useState(existingQuery?.tags ?? '')

  const createMutation = useCreateSavedQuery()
  const updateMutation = useUpdateSavedQuery()

  const handleSave = () => {
    if (!title.trim()) return

    if (existingQuery) {
      updateMutation.mutate(
        { id: existingQuery.id, title, description, sql, database: database ?? '', tags },
        {
          onSuccess: () => {
            onSaved?.()
            onClose()
          },
        },
      )
    } else {
      createMutation.mutate(
        { title, description, sql, database: database ?? '', tags },
        {
          onSuccess: () => {
            onSaved?.()
            onClose()
          },
        },
      )
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending
  const error = createMutation.error || updateMutation.error

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="w-full max-w-md animate-slide-up rounded-xl border border-surface-200 bg-white p-5 shadow-lg shadow-black/5 dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/30"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-sm font-semibold text-gray-800 dark:text-gray-200">
          {existingQuery ? 'Update Query' : 'Save Query'}
        </h2>

        <label className="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Title</label>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="mb-3 block w-full rounded border border-gray-300 bg-white px-2.5 py-1.5 text-sm focus:border-accent-400 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-200"
          placeholder="My query"
          autoFocus
          onKeyDown={(e) => e.key === 'Enter' && handleSave()}
        />

        <label className="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Description</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className="mb-3 block w-full rounded border border-gray-300 bg-white px-2.5 py-1.5 text-sm focus:border-accent-400 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-200"
          rows={2}
          placeholder="Optional description"
        />

        <label className="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Tags</label>
        <input
          type="text"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          className="mb-3 block w-full rounded border border-gray-300 bg-white px-2.5 py-1.5 text-sm focus:border-accent-400 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-200"
          placeholder="analytics, users (comma-separated)"
        />

        <div className="mb-3 rounded border border-surface-200 bg-surface-50 p-2 dark:border-surface-800 dark:bg-surface-800">
          <pre className="max-h-24 overflow-auto text-xs text-gray-600 dark:text-gray-400">{sql}</pre>
        </div>

        {error && (
          <p className="mb-3 text-xs text-danger">{error.message}</p>
        )}

        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-surface-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-surface-800"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isPending || !title.trim()}
            className="rounded bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50"
          >
            {isPending ? 'Saving...' : existingQuery ? 'Update' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
