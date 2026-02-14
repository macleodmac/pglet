import { useDatabases, useSwitchDatabase } from '../../api/queries'
import { useConnectionStore } from '../../stores/connection'
import type { ConnectionInfo } from '../../api/generated/types.gen'

/**
 * DatabaseSwitcher Component
 *
 * A dropdown for switching between available databases in the current PostgreSQL connection.
 *
 * Features:
 * - Fetches available databases using useDatabases()
 * - Displays current database from connection store
 * - Switches database on selection and updates connection store
 * - Auto-hides when only one database is available
 * - Disabled during database switch operation
 *
 * Usage:
 * ```tsx
 * <DatabaseSwitcher />
 * ```
 */
export function DatabaseSwitcher() {
  const { data: databases } = useDatabases()
  const currentDb = useConnectionStore((s) => s.info?.database)
  const setConnected = useConnectionStore((s) => s.setConnected)
  const switchDb = useSwitchDatabase()

  const handleSwitch = (db: string) => {
    if (db === currentDb) return
    switchDb.mutate(db, {
      onSuccess: (info: ConnectionInfo) => {
        setConnected(info)
      },
    })
  }

  // Hide if no databases or only one database
  if (!databases || databases.length <= 1) return null

  return (
    <select
      value={currentDb ?? ''}
      onChange={(e) => handleSwitch(e.target.value)}
      disabled={switchDb.isPending}
      className="rounded border border-gray-300 bg-white px-2 py-0.5 text-xs text-gray-700 transition-colors focus:border-accent-400 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-300 dark:focus:border-accent-400"
      aria-label="Switch database"
    >
      {databases.map((db) => (
        <option key={db} value={db}>
          {db}
        </option>
      ))}
    </select>
  )
}
