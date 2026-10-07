import { useConnectionStore } from '../../stores/connection'
import { useSetConnectionOptions } from '../../api/queries'

const TIMEOUTS = [
  { ms: 0, label: 'No timeout' },
  { ms: 5_000, label: '5s timeout' },
  { ms: 30_000, label: '30s timeout' },
  { ms: 60_000, label: '1m timeout' },
  { ms: 300_000, label: '5m timeout' },
]

/**
 * Status bar controls for the session's read-only mode and statement timeout.
 * Changing either reconnects, which the server refuses while a transaction is open.
 */
export function SessionOptions() {
  const info = useConnectionStore((s) => s.info)
  const setConnected = useConnectionStore((s) => s.setConnected)
  const setOptions = useSetConnectionOptions()

  if (!info) return null
  const readOnly = !!info.read_only
  const timeoutMs = info.statement_timeout_ms ?? 0

  const apply = (opts: { read_only: boolean; statement_timeout_ms: number }) =>
    setOptions.mutate(opts, { onSuccess: setConnected })

  // Keep a non-preset timeout (e.g. from --statement-timeout) selectable.
  const timeouts = TIMEOUTS.some((t) => t.ms === timeoutMs)
    ? TIMEOUTS
    : [...TIMEOUTS, { ms: timeoutMs, label: `${timeoutMs / 1000}s timeout` }]

  return (
    <>
      <button
        type="button"
        onClick={() => apply({ read_only: !readOnly, statement_timeout_ms: timeoutMs })}
        disabled={setOptions.isPending}
        className={`rounded px-1.5 py-px text-[11px] font-medium disabled:opacity-50 ${
          readOnly
            ? 'bg-sky-500/15 text-sky-700 hover:bg-sky-500/25 dark:text-sky-400'
            : 'text-gray-500 hover:bg-surface-200 dark:text-gray-400 dark:hover:bg-surface-800'
        }`}
        title={
          readOnly
            ? 'Read-only: writes are rejected. Click to allow writes.'
            : 'Writes allowed. Click to make the session read-only.'
        }
      >
        {readOnly ? 'Read-only' : 'Read-write'}
      </button>
      <select
        value={timeoutMs}
        onChange={(e) => apply({ read_only: readOnly, statement_timeout_ms: Number(e.target.value) })}
        disabled={setOptions.isPending}
        className="rounded border border-gray-300 bg-white px-1 py-0 text-xs text-gray-700 focus:border-accent-400 focus:outline-none disabled:opacity-50 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-300"
        aria-label="Statement timeout"
        title="Cancel statements that run longer than this"
      >
        {timeouts.map((t) => (
          <option key={t.ms} value={t.ms}>
            {t.label}
          </option>
        ))}
      </select>
      {setOptions.error && (
        <span className="truncate text-danger" title={setOptions.error.message}>
          {setOptions.error.message}
        </span>
      )}
    </>
  )
}
