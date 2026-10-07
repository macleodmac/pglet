import type { QueryResult } from '../../api/generated'

const PAST_TENSE: Record<string, string> = {
  INSERT: 'inserted',
  UPDATE: 'updated',
  DELETE: 'deleted',
  MERGE: 'merged',
  COPY: 'copied',
}

function plural(n: number) {
  return `${n.toLocaleString()} ${n === 1 ? 'row' : 'rows'}`
}

/** Short stats-bar summary, e.g. "3 rows updated", "120 rows", "CREATE TABLE". */
export function resultSummary(result: QueryResult): string {
  const tag = result.command_tag ?? ''
  const verb = PAST_TENSE[tag]
  if (verb && result.rows_affected != null) return `${plural(result.rows_affected)} ${verb}`
  if (result.truncated) {
    return result.rows_affected != null
      ? `${result.row_count.toLocaleString()} of ${plural(result.rows_affected)}`
      : `first ${plural(result.row_count)}`
  }
  if (result.columns.length > 0) return plural(result.row_count)
  return tag || 'OK'
}

/** Body shown for commands that return no result set (DML without RETURNING, DDL, ...). */
export function CommandResult({ result }: { result: QueryResult }) {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="flex flex-col items-center gap-1 px-6 py-8 text-center">
        {result.command_tag && (
          <span className="font-mono text-[11px] uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {result.command_tag}
          </span>
        )}
        <span className="text-lg font-medium text-gray-700 dark:text-gray-300">
          {resultSummary(result)}
        </span>
        {result.in_transaction && (
          <span className="text-xs text-amber-600 dark:text-amber-400">Not committed. The transaction is still open.</span>
        )}
      </div>
    </div>
  )
}
