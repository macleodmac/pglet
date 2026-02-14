import { useConnectionStore } from '../../stores/connection'
import { DatabaseSwitcher } from '../connection/DatabaseSwitcher'

export function StatusBar() {
  const info = useConnectionStore((s) => s.info)

  return (
    <div className="flex h-6 items-center gap-3 border-t border-surface-200 bg-surface-50 px-3 font-mono text-xs text-gray-500 dark:border-surface-800 dark:bg-surface-900 dark:text-gray-400">
      {info ? (
        <>
          <span className="flex items-center gap-1.5">
            <span className="h-1.5 w-1.5 rounded-full bg-green-500" />
            Connected: {info.database}@{info.host}:{info.port}
          </span>
          <DatabaseSwitcher />
          <span className="text-gray-300 dark:text-gray-600">|</span>
          <span>PG {info.version}</span>
        </>
      ) : (
        <span>Not connected</span>
      )}
    </div>
  )
}
