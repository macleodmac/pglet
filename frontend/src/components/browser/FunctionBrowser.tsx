import Editor from '@monaco-editor/react'
import { useFunctionDefinition } from '../../api/queries'
import { useThemeStore } from '../../stores/theme'

export function FunctionBrowser({ functionName }: { functionName: string }) {
  const { data, isLoading, error } = useFunctionDefinition(functionName)
  const theme = useThemeStore((s) => s.theme)

  if (isLoading) {
    return <div className="p-4 text-sm text-gray-400">Loading...</div>
  }

  if (error || !data) {
    return (
      <div className="p-4 text-sm text-danger">
        {error?.message || 'Function not found'}
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Info bar */}
      <div className="flex flex-shrink-0 items-center gap-3 border-b border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900">
        <span className="text-xs font-medium text-gray-700 dark:text-gray-300">
          {data.schema}.{data.name}
        </span>
        <span className="rounded bg-surface-200 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-surface-700 dark:text-gray-400">
          {data.kind}
        </span>
        <span className="rounded bg-surface-200 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-surface-700 dark:text-gray-400">
          {data.language}
        </span>
        <span className="rounded bg-surface-200 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-surface-700 dark:text-gray-400">
          {data.volatility}
        </span>
        {data.return_type && (
          <span className="text-xs text-gray-500 dark:text-gray-400">
            returns {data.return_type}
          </span>
        )}
      </div>

      {/* Source code */}
      <div className="min-h-0 flex-1">
        <Editor
          language="pgsql"
          value={data.definition}
          theme={theme === 'dark' ? 'vs-dark' : 'light'}
          options={{
            readOnly: true,
            minimap: { enabled: false },
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            fontSize: 13,
            padding: { top: 8 },
          }}
        />
      </div>
    </div>
  )
}
