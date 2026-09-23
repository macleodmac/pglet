import { useState, useCallback, useEffect } from 'react'
import type { Tab } from '../../stores/tabs'
import { useTabStore } from '../../stores/tabs'
import {
  useRunQuery,
  useExplainQuery,
  useCancelQuery,
  useBeginTransaction,
  useCommitTransaction,
  useRollbackTransaction,
  autoNameTab,
} from '../../api/queries'
import type { QueryResult } from '../../api/generated'

function errorResult(message: string): QueryResult {
  return {
    columns: [],
    column_types: [],
    rows: [],
    row_count: 0,
    duration_ms: 0,
    error: message,
  }
}

export function useTabExecution(tab: Tab) {
  const updateTab = useTabStore((s) => s.updateTab)
  const runQuery = useRunQuery()
  const explainQuery = useExplainQuery()
  const cancelQuery = useCancelQuery()
  const beginTx = useBeginTransaction()
  const commitTx = useCommitTransaction()
  const rollbackTx = useRollbackTransaction()
  const [result, setResult] = useState<QueryResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  const run = useCallback(
    (sql?: string) => {
      const query = sql || tab.sql
      if (!query.trim()) return
      setIsRunning(true)
      runQuery.mutate(
        { query, tab_id: tab.id },
        {
          onSuccess: (data) => {
            setResult(data)
            setIsRunning(false)
            // A query can open or abort a transaction, e.g. an error inside one leaves it open.
            updateTab(tab.id, { inTransaction: !!data.in_transaction })
            if (!tab.userRenamed && !tab.aiRenamed && !data.error) {
              updateTab(tab.id, { aiRenamed: true })
              autoNameTab(query).then((name) => {
                if (name) updateTab(tab.id, { title: name, aiRenamed: true })
              })
            }
          },
          onError: (err) => {
            setResult(errorResult(err.message))
            setIsRunning(false)
          },
        },
      )
    },
    [tab.sql, tab.id, tab.userRenamed, tab.aiRenamed, runQuery, updateTab],
  )

  const explain = useCallback(() => {
    if (!tab.sql.trim()) return
    setIsRunning(true)
    explainQuery.mutate(
      { query: tab.sql, tab_id: tab.id },
      {
        onSuccess: (data) => {
          setResult(data)
          setIsRunning(false)
        },
        onError: (err) => {
          setResult(errorResult(err.message))
          setIsRunning(false)
        },
      },
    )
  }, [tab.sql, tab.id, explainQuery])

  const cancel = useCallback(() => {
    cancelQuery.mutate(tab.id)
  }, [tab.id, cancelQuery])

  const syncTransaction = useCallback(
    (data: { in_transaction: boolean }) => updateTab(tab.id, { inTransaction: data.in_transaction }),
    [tab.id, updateTab],
  )

  const begin = useCallback(() => {
    beginTx.mutate(tab.id, { onSuccess: syncTransaction })
  }, [tab.id, beginTx, syncTransaction])

  const commit = useCallback(() => {
    commitTx.mutate(tab.id, { onSuccess: syncTransaction })
  }, [tab.id, commitTx, syncTransaction])

  const rollback = useCallback(() => {
    rollbackTx.mutate(tab.id, { onSuccess: syncTransaction })
  }, [tab.id, rollbackTx, syncTransaction])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isRunning) {
        cancel()
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [isRunning, cancel])

  return {
    result,
    isRunning,
    txPending: beginTx.isPending || commitTx.isPending || rollbackTx.isPending,
    run,
    explain,
    cancel,
    begin,
    commit,
    rollback,
  }
}

export type TabExecution = ReturnType<typeof useTabExecution>
