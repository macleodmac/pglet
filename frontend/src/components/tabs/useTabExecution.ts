import { useState, useCallback, useEffect } from 'react'
import type { Tab } from '../../stores/tabs'
import { useTabStore } from '../../stores/tabs'
import { useSettingsStore } from '../../stores/settings'
import {
  useAiFix,
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
  const appendAiTurn = useTabStore((s) => s.appendAiTurn)
  const safeWrites = useSettingsStore((s) => s.safeWrites)
  const runQuery = useRunQuery()
  const explainQuery = useExplainQuery()
  const cancelQuery = useCancelQuery()
  const beginTx = useBeginTransaction()
  const commitTx = useCommitTransaction()
  const rollbackTx = useRollbackTransaction()
  const aiFix = useAiFix()
  const [result, setResult] = useState<QueryResult | null>(null)
  /** SQL that produced `result`, which may be a selection of the editor */
  const [resultSql, setResultSql] = useState('')
  /** Result of the write that safe writes is holding for review */
  const [heldWrite, setHeldWrite] = useState<QueryResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  const syncTransaction = useCallback(
    (data: { in_transaction?: boolean }) => {
      updateTab(tab.id, { inTransaction: !!data.in_transaction })
      if (!data.in_transaction) setHeldWrite(null)
    },
    [tab.id, updateTab],
  )

  const run = useCallback(
    (sql?: string) => {
      const query = sql || tab.sql
      if (!query.trim()) return
      setIsRunning(true)
      setResultSql(query)
      aiFix.reset()
      runQuery.mutate(
        { query, tab_id: tab.id, safe_writes: safeWrites },
        {
          onSuccess: (data) => {
            setResult(data)
            setIsRunning(false)
            // A query can open or abort a transaction, e.g. an error inside one leaves it open.
            syncTransaction(data)
            if (data.held_for_review) setHeldWrite(data)
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
    [tab.sql, tab.id, tab.userRenamed, tab.aiRenamed, safeWrites, runQuery, aiFix, updateTab, syncTransaction],
  )

  const explain = useCallback(() => {
    if (!tab.sql.trim()) return
    setIsRunning(true)
    setResultSql(tab.sql)
    aiFix.reset()
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
  }, [tab.sql, tab.id, explainQuery, aiFix])

  const cancel = useCallback(() => {
    cancelQuery.mutate(tab.id)
  }, [tab.id, cancelQuery])

  const begin = useCallback(() => {
    beginTx.mutate(tab.id, { onSuccess: syncTransaction })
  }, [tab.id, beginTx, syncTransaction])

  const commit = useCallback(() => {
    commitTx.mutate(tab.id, { onSuccess: syncTransaction })
  }, [tab.id, commitTx, syncTransaction])

  const rollback = useCallback(() => {
    rollbackTx.mutate(tab.id, { onSuccess: syncTransaction })
  }, [tab.id, rollbackTx, syncTransaction])

  const requestFix = useCallback(() => {
    if (!result?.error) return
    aiFix.mutate({ sql: resultSql, error: result.error })
  }, [result, resultSql, aiFix])

  const applyFix = useCallback(
    (andRun: boolean) => {
      const fix = aiFix.data
      if (!fix) return
      if (tab.type === 'ai') {
        const firstLine = result?.error?.split('\n')[0].replace(/^ERROR:\s+/, '') ?? ''
        appendAiTurn(tab.id, { prompt: `Fix: ${firstLine}`, sql: fix.sql, explanation: fix.explanation })
      } else {
        // The failed SQL may have been a selection, so replace only that part.
        const sql = tab.sql.includes(resultSql) ? tab.sql.replace(resultSql, () => fix.sql) : fix.sql
        updateTab(tab.id, { sql })
      }
      aiFix.reset()
      if (andRun) run(fix.sql)
    },
    [aiFix, tab.type, tab.id, tab.sql, result, resultSql, appendAiTurn, updateTab, run],
  )

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
    heldWrite: tab.inTransaction ? heldWrite : null,
    fix: aiFix,
    requestFix,
    applyFix,
  }
}

export type TabExecution = ReturnType<typeof useTabExecution>
