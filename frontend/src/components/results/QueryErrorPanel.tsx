import { highlightSql } from '../ai/highlightSql'
import { Icon } from '../ui/Icon'

export interface SqlFix {
  sql: string
  explanation: string
}

// Errors the query text can't cause: connection loss, transaction state,
// cancellation and server resource problems.
const UNFIXABLE_SQLSTATE = /^(08|25|53|57|58)/

export function isFixableError(code: string | undefined): boolean {
  return !!code && !UNFIXABLE_SQLSTATE.test(code)
}

interface QueryErrorPanelProps {
  error: string
  canFix: boolean
  isFixing: boolean
  fix: SqlFix | null
  fixError: string | null
  onFix: () => void
  onApply: (run: boolean) => void
  onDismiss: () => void
}

export function QueryErrorPanel({
  error,
  canFix,
  isFixing,
  fix,
  fixError,
  onFix,
  onApply,
  onDismiss,
}: QueryErrorPanelProps) {
  return (
    <div className="space-y-3 p-3">
      <pre className="whitespace-pre-wrap font-mono text-xs leading-relaxed text-danger">{error}</pre>

      {canFix && !fix && (
        <button
          type="button"
          onClick={onFix}
          disabled={isFixing}
          className="flex items-center gap-1 rounded border border-accent-400/50 px-2 py-1 text-xs font-medium text-accent-600 hover:bg-accent-500/10 disabled:opacity-60 dark:text-accent-400"
        >
          <Icon name="wand" className={`h-3 w-3 ${isFixing ? 'animate-pulse' : ''}`} />
          {isFixing ? 'Fixing…' : 'Fix with AI'}
        </button>
      )}

      {fixError && <p className="text-xs text-danger">{fixError}</p>}

      {fix && (
        <div className="animate-fade-in rounded-md border border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-900">
          <div className="flex items-start gap-1.5 px-3 pt-2.5">
            <Icon name="lightbulb" className="mt-0.5 h-2.5 w-2.5 flex-shrink-0 text-accent-500" />
            <p className="text-xs leading-relaxed text-gray-600 dark:text-gray-400">{fix.explanation}</p>
          </div>
          <pre
            className="mx-3 my-2.5 overflow-x-auto whitespace-pre-wrap rounded border border-surface-200 bg-white px-2.5 py-2 font-mono text-xs text-gray-800 dark:border-surface-800 dark:bg-surface-950 dark:text-gray-200"
            dangerouslySetInnerHTML={{ __html: highlightSql(fix.sql) }}
          />
          <div className="flex items-center gap-1.5 border-t border-surface-200 px-3 py-1.5 dark:border-surface-800">
            <button
              type="button"
              onClick={() => onApply(true)}
              className="flex items-center gap-1 rounded bg-accent-500 px-2.5 py-1 text-xs font-medium text-white hover:bg-accent-600"
            >
              <Icon name="play" className="h-2.5 w-2.5" />
              Apply &amp; run
            </button>
            <button
              type="button"
              onClick={() => onApply(false)}
              className="rounded border border-surface-300 px-2.5 py-1 text-xs font-medium text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
            >
              Apply
            </button>
            <button
              type="button"
              onClick={onDismiss}
              className="ml-auto rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
