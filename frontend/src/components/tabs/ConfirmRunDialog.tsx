import { useEffect, useRef } from 'react'
import type { DestructiveStatement } from '../../lib/sqlSafety'

function preview(sql: string) {
  const oneLine = sql.replace(/\s+/g, ' ')
  return oneLine.length > 160 ? `${oneLine.slice(0, 157)}...` : oneLine
}

/** Asks before running statements that can destroy data. Cancel has focus. */
export function ConfirmRunDialog({
  statements,
  inTransaction,
  onConfirm,
  onCancel,
}: {
  statements: DestructiveStatement[]
  inTransaction: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const cancelRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    cancelRef.current?.focus()
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onCancel])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onCancel}>
      <div
        role="alertdialog"
        aria-labelledby="confirm-run-title"
        className="w-full max-w-lg animate-slide-up rounded-xl border border-surface-200 bg-white p-5 shadow-lg shadow-black/5 dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/30"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id="confirm-run-title" className="mb-1 text-sm font-semibold text-gray-800 dark:text-gray-200">
          Run {statements.length === 1 ? 'this statement' : `these ${statements.length} statements`}?
        </h2>
        <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
          {inTransaction
            ? 'This tab has an open transaction, so you can still roll back afterwards.'
            : 'There is no open transaction, so this takes effect immediately.'}
        </p>

        <ul className="mb-4 max-h-60 space-y-2 overflow-auto">
          {statements.map((s, i) => (
            <li key={i} className="rounded-md border border-danger/30 bg-danger/5 px-3 py-2">
              <p className="mb-1 text-xs font-medium text-danger">{s.reason}</p>
              <code className="block break-all font-mono text-[11px] text-gray-700 dark:text-gray-300">
                {preview(s.sql)}
              </code>
            </li>
          ))}
        </ul>

        <div className="flex justify-end gap-2">
          <button
            ref={cancelRef}
            type="button"
            onClick={onCancel}
            className="rounded-md border border-surface-300 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="rounded-md bg-danger px-3 py-1.5 text-xs font-medium text-white hover:bg-danger/90"
          >
            Run anyway
          </button>
        </div>
      </div>
    </div>
  )
}
