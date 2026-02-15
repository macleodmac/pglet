import { useEffect, useRef, useState } from 'react'

import { JsonbViewer } from './JsonbViewer'

interface CellDetailProps {
  column: string
  columnType: string
  value: string | null
  onClose: () => void
}

function isJson(value: string): boolean {
  if (!value) return false
  const trimmed = value.trim()
  if ((trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
    try {
      JSON.parse(trimmed)
      return true
    } catch {
      return false
    }
  }
  return false
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

const uuidSegmentColors = [
  'text-blue-600 dark:text-blue-400',
  'text-purple-600 dark:text-purple-400',
  'text-emerald-600 dark:text-emerald-400',
  'text-amber-600 dark:text-amber-400',
  'text-rose-600 dark:text-rose-400',
] as const

function UuidValue({ value }: { value: string }) {
  const segments = value.split('-')
  return (
    <span className="font-mono text-sm">
      {segments.map((seg, i) => (
        <span key={seg}>
          {i > 0 && <span className="text-gray-300 dark:text-gray-600">-</span>}
          <span className={uuidSegmentColors[i]}>{seg}</span>
        </span>
      ))}
    </span>
  )
}

const TIMESTAMP_TYPES = new Set(['TIMESTAMP', 'TIMESTAMPTZ', 'TIMESTAMP WITHOUT TIME ZONE', 'TIMESTAMP WITH TIME ZONE'])

function isTimestampType(columnType: string): boolean {
  return TIMESTAMP_TYPES.has(columnType.toUpperCase())
}

function TimestampValue({ value }: { value: string }) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return <span className="font-mono text-sm text-gray-800 dark:text-gray-200">{value}</span>

  const dateStr = date.toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' })
  const timeStr = date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  const msStr = date.getMilliseconds() > 0 ? `.${String(date.getMilliseconds()).padStart(3, '0')}` : ''
  const tz = Intl.DateTimeFormat(undefined, { timeZoneName: 'short' }).formatToParts(date).find(p => p.type === 'timeZoneName')?.value

  return (
    <span className="font-mono text-sm">
      <span className="text-blue-600 dark:text-blue-400">{dateStr}</span>
      <span className="text-gray-300 dark:text-gray-600">{' · '}</span>
      <span className="text-emerald-600 dark:text-emerald-400">{timeStr}{msStr}</span>
      {tz && (
        <>
          <span className="text-gray-300 dark:text-gray-600">{' · '}</span>
          <span className="text-purple-600 dark:text-purple-400">{tz}</span>
        </>
      )}
    </span>
  )
}

function formatBytes(str: string): string {
  const bytes = new Blob([str]).size
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function CellDetail({ column, columnType, value, onClose }: CellDetailProps) {
  const showJson = value !== null && isJson(value)
  const showUuid = value !== null && UUID_RE.test(value)
  const showTimestamp = value !== null && !showJson && !showUuid && isTimestampType(columnType)
  const [copied, setCopied] = useState(false)
  const copiedTimeout = useRef<ReturnType<typeof setTimeout>>(null)
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  // Focus the panel for accessibility
  useEffect(() => {
    panelRef.current?.focus()
  }, [])

  const handleCopy = () => {
    if (value === null) return
    navigator.clipboard.writeText(value).then(() => {
      setCopied(true)
      if (copiedTimeout.current) clearTimeout(copiedTimeout.current)
      copiedTimeout.current = setTimeout(() => setCopied(false), 1500)
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-[2px]">
      <button
        type="button"
        aria-label="Close dialog"
        className="absolute inset-0 cursor-default"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        tabIndex={-1}
        role="dialog"
        aria-label={`Cell detail: ${column}`}
        className="relative z-10 flex max-h-[80vh] w-full max-w-2xl animate-slide-up flex-col overflow-hidden rounded-xl border border-surface-200 bg-white shadow-xl shadow-black/8 outline-none dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/40"
      >
        {/* Header */}
        <div className="flex items-center gap-3 border-b border-surface-200 px-4 py-3 dark:border-surface-800">
          <div className="flex min-w-0 flex-1 items-center gap-2.5">
            <span className="truncate font-mono text-sm font-medium text-gray-800 dark:text-gray-200">
              {column}
            </span>
            <span className="shrink-0 rounded-md border border-surface-200 bg-surface-50 px-1.5 py-0.5 font-mono text-[10px] tracking-wide text-gray-400 dark:border-surface-800 dark:bg-surface-800 dark:text-gray-500">
              {columnType}
            </span>
          </div>

          <div className="flex items-center gap-1">
            {/* Copy button */}
            {value !== null && (
              <button
                type="button"
                onClick={handleCopy}
                className={`rounded-md p-1.5 transition-colors ${
                  copied
                    ? 'text-success'
                    : 'text-gray-400 hover:bg-surface-100 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300'
                }`}
                title={copied ? 'Copied' : 'Copy value'}
              >
                {copied ? (
                  <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <title>Copied</title>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                ) : (
                  <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <title>Copy</title>
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                  </svg>
                )}
              </button>
            )}

            {/* Close button */}
            <button
              type="button"
              onClick={onClose}
              className="rounded-md p-1.5 text-gray-400 hover:bg-surface-100 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
              title="Close (Esc)"
            >
              <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <title>Close</title>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Value area */}
        <div className="overflow-auto p-4">
          <div className="rounded-lg border border-surface-200 bg-surface-50 p-3 dark:border-surface-800 dark:bg-surface-800/60">
            {value === null ? (
              <span className="font-mono text-sm italic text-gray-300 dark:text-gray-600">NULL</span>
            ) : showJson ? (
              <JsonbViewer data={JSON.parse(value)} />
            ) : showUuid ? (
              <UuidValue value={value} />
            ) : showTimestamp ? (
              <TimestampValue value={value} />
            ) : (
              <pre className="whitespace-pre-wrap break-all font-mono text-sm leading-relaxed text-gray-800 dark:text-gray-200">
                {value}
              </pre>
            )}
          </div>
        </div>

        {/* Footer — size hint */}
        {value !== null && value.length > 0 && (
          <div className="border-t border-surface-200 px-4 py-2 dark:border-surface-800">
            <span className="font-mono text-[10px] tracking-wide text-gray-400 dark:text-gray-500">
              {value.length.toLocaleString()} chars · {formatBytes(value)}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
