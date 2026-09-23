import type { RefObject } from 'react'
import type { SqlEditorHandle } from '../editor/SqlEditor'
import { DataGrid } from '../results/DataGrid'
import { ExplainPlan, isExplainResult } from '../results/ExplainPlan'
import { QueryResultsEmpty } from '../results/QueryResultsEmpty'
import { CommandResult, resultSummary } from '../results/CommandResult'
import { Icon } from '../ui/Icon'
import type { TabExecution } from './useTabExecution'

async function downloadExport(query: string, format: 'csv' | 'json') {
  const res = await fetch('/api/export', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, format }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'Export failed')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `export.${format}`
  document.body.appendChild(a)
  a.click()
  URL.revokeObjectURL(url)
  a.remove()
}

export function ResultsPanel({
  sql,
  inTransaction,
  execution,
  editorRef,
  onSave,
}: {
  sql: string
  inTransaction: boolean
  execution: TabExecution
  editorRef: RefObject<SqlEditorHandle | null>
  onSave: () => void
}) {
  const { result, isRunning, txPending } = execution
  const showStats = result && !result.error && !isExplainResult(result.columns)

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-shrink-0 items-center gap-1.5 border-y border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900">
        {isRunning ? (
          <button
            type="button"
            onClick={execution.cancel}
            className="flex items-center gap-1 rounded bg-danger px-3 py-1 text-xs font-medium text-white hover:bg-danger/90"
          >
            <Icon name="xmark" className="h-3 w-3" />
            Cancel
          </button>
        ) : (
          <button
            type="button"
            onClick={() => execution.run(editorRef.current?.getSelectedText())}
            className="flex items-center gap-1 rounded bg-accent-500 px-3 py-1 text-xs font-medium text-white hover:bg-accent-600"
          >
            <Icon name="play" className="h-3 w-3" />
            Run
          </button>
        )}
        <button
          type="button"
          onClick={execution.explain}
          disabled={isRunning}
          className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs font-medium text-gray-700 hover:bg-surface-100 disabled:opacity-50 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
        >
          <Icon name="lightbulb" className="h-3 w-3" />
          Explain
        </button>

        <span className="mx-0.5 h-3.5 w-px bg-surface-300 dark:bg-surface-700" />
        <TransactionControls
          inTransaction={inTransaction}
          disabled={isRunning || txPending}
          onBegin={execution.begin}
          onCommit={execution.commit}
          onRollback={execution.rollback}
        />

        {showStats && (
          <span className="ml-2 font-mono text-[11px] text-gray-400 dark:text-gray-500">
            {resultSummary(result)} · {result.duration_ms}ms
          </span>
        )}

        <div className="ml-auto flex items-center gap-1.5">
          {showStats && (
            <>
              <button
                type="button"
                onClick={() => downloadExport(sql, 'csv')}
                className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                title="Export as CSV"
              >
                <Icon name="file-csv" className="h-3 w-3" />
                CSV
              </button>
              <button
                type="button"
                onClick={() => downloadExport(sql, 'json')}
                className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                title="Export as JSON"
              >
                <Icon name="file-code" className="h-3 w-3" />
                JSON
              </button>
              <span className="mx-0.5 h-3.5 w-px bg-surface-300 dark:bg-surface-700" />
            </>
          )}
          <button
            type="button"
            onClick={() => editorRef.current?.formatSql()}
            className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
            title="Format SQL (Alt+Shift+F)"
          >
            <Icon name="wand" className="h-3 w-3" />
            Format
          </button>
          <button
            type="button"
            onClick={onSave}
            className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
            title="Save query (Cmd+S)"
          >
            <Icon name="save" className="h-3 w-3" />
            Save
          </button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
        {result?.error ? (
          <div className="p-3 text-sm text-danger dark:text-danger">{result.error}</div>
        ) : result && isExplainResult(result.columns) ? (
          <ExplainPlan rows={result.rows} durationMs={result.duration_ms} />
        ) : result && result.columns.length === 0 ? (
          <CommandResult result={result} />
        ) : result ? (
          <DataGrid columns={result.columns} columnTypes={result.column_types} rows={result.rows} />
        ) : (
          <QueryResultsEmpty />
        )}
      </div>
    </div>
  )
}

const txButtonBase =
  'flex items-center gap-1 rounded border px-2 py-1 text-xs font-medium disabled:opacity-50'

function TransactionControls({
  inTransaction,
  disabled,
  onBegin,
  onCommit,
  onRollback,
}: {
  inTransaction: boolean
  disabled: boolean
  onBegin: () => void
  onCommit: () => void
  onRollback: () => void
}) {
  if (!inTransaction) {
    return (
      <button
        type="button"
        onClick={onBegin}
        disabled={disabled}
        className={`${txButtonBase} border-surface-300 text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800`}
        title="Begin a transaction (BEGIN)"
      >
        <Icon name="code-branch" className="h-3 w-3" />
        Begin
      </button>
    )
  }

  return (
    <>
      <span
        className="flex items-center gap-1.5 rounded bg-amber-400/15 px-2 py-1 text-xs font-medium text-amber-600 dark:text-amber-400"
        title="This tab has an open transaction"
      >
        <span className="h-1.5 w-1.5 rounded-full bg-amber-500" />
        In transaction
      </span>
      <button
        type="button"
        onClick={onCommit}
        disabled={disabled}
        className={`${txButtonBase} border-success/40 text-success hover:bg-success/10`}
        title="Commit the transaction (COMMIT)"
      >
        <Icon name="check" className="h-3 w-3" />
        Commit
      </button>
      <button
        type="button"
        onClick={onRollback}
        disabled={disabled}
        className={`${txButtonBase} border-danger/40 text-danger hover:bg-danger/10`}
        title="Roll back the transaction (ROLLBACK)"
      >
        <Icon name="rotate-left" className="h-3 w-3" />
        Rollback
      </button>
    </>
  )
}
