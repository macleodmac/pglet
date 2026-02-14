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

export function CellDetail({ column, columnType, value, onClose }: CellDetailProps) {
  const showJson = value !== null && isJson(value)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="max-h-[80vh] w-full max-w-2xl animate-slide-up overflow-auto rounded-xl border border-surface-200 bg-white p-4 shadow-lg shadow-black/5 dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/30"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between">
          <div>
            <span className="text-sm font-semibold text-gray-800 dark:text-gray-200">{column}</span>
            <span className="ml-2 text-xs text-gray-400">{columnType}</span>
          </div>
          <button
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:bg-surface-100 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
          >
            ✕
          </button>
        </div>
        <div className="rounded border border-surface-200 bg-surface-50 p-3 dark:border-surface-800 dark:bg-surface-800">
          {value === null ? (
            <span className="italic text-gray-400">NULL</span>
          ) : showJson ? (
            <JsonbViewer data={JSON.parse(value)} />
          ) : (
            <pre className="font-mono whitespace-pre-wrap break-all text-sm text-gray-800 dark:text-gray-200">
              {value}
            </pre>
          )}
        </div>
      </div>
    </div>
  )
}
