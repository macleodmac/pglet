import { useState, useCallback, useEffect } from 'react'
import type { Tab } from '../../stores/tabs'
import { useTabStore } from '../../stores/tabs'
import { useSettingsStore } from '../../stores/settings'
import { useConnectionStore } from '../../stores/connection'
import { findDestructiveStatements, type DestructiveStatement } from '../../lib/sqlSafety'
import {
  useRunQuery,
  useExplainQuery,
  useAnalyzeQuery,
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

/** A run waiting for the user to confirm destructive statements. */
export interface PendingConfirm {
  query: string
  statements: DestructiveStatement[]
}

export function useTabExecution(tab: Tab) {
  const updateTab = useTabStore((s) => s.updateTab)
  const maxRows = useSettingsStore((s) => s.maxRows)
  const readOnly = useConnectionStore((s) => !!s.info?.read_only)
  const runQuery = useRunQuery()
  const explainQuery = useExplainQuery()
  const analyzeQuery = useAnalyzeQuery()
  const cancelQuery = useCancelQuery()
  const beginTx = useBeginTransaction()
  const commitTx = useCommitTransaction()
  const rollbackTx = useRollbackTransaction()
  const [result, setResult] = useState<QueryResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)
  const [pendingConfirm, setPendingConfirm] = useState<PendingConfirm | null>(null)

  const execute = useCallback(
    (query: string) => {
      setIsRunning(true)
      runQuery.mutate(
        { query, tab_id: tab.id, max_rows: maxRows },
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
    [tab.id, tab.userRenamed, tab.aiRenamed, maxRows, runQuery, updateTab],
  )

  /** Runs sql (or the whole tab), asking first if it looks destructive. */
  const run = useCallback(
    (sql?: string) => {
      const query = sql || tab.sql
      if (!query.trim()) return
      // A read-only session rejects writes itself, so there is nothing to confirm.
      const statements = readOnly ? [] : findDestructiveStatements(query)
      if (statements.length > 0) {
        setPendingConfirm({ query, statements })
        return
      }
      execute(query)
    },
    [tab.sql, readOnly, execute],
  )

  const confirmRun = useCallback(() => {
    if (!pendingConfirm) return
    setPendingConfirm(null)
    execute(pendingConfirm.query)
  }, [pendingConfirm, execute])

  const dismissConfirm = useCallback(() => setPendingConfirm(null), [])

  const plan = useCallback(
    (mutation: typeof explainQuery, sql?: string) => {
      const query = sql || tab.sql
      if (!query.trim()) return
      setIsRunning(true)
      mutation.mutate(
        { query, tab_id: tab.id },
        {
          onSuccess: (data) => {
            setResult(data)
            setIsRunning(false)
            // An error inside a tab's transaction is contained by a savepoint,
            // but keep the indicator in sync all the same.
            updateTab(tab.id, { inTransaction: !!data.in_transaction })
          },
          onError: (err) => {
            setResult(errorResult(err.message))
            setIsRunning(false)
          },
        },
      )
    },
    [tab.sql, tab.id, updateTab],
  )

  /** EXPLAIN the selection or tab without running it. */
  const explain = useCallback((sql?: string) => plan(explainQuery, sql), [plan, explainQuery])
  /** EXPLAIN ANALYZE: runs the statement, then rolls it back. */
  const analyze = useCallback((sql?: string) => plan(analyzeQuery, sql), [plan, analyzeQuery])

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
    pendingConfirm,
    inTransaction: !!tab.inTransaction,
    run,
    confirmRun,
    dismissConfirm,
    explain,
    analyze,
    cancel,
    begin,
    commit,
    rollback,
  }
}

export type TabExecution = ReturnType<typeof useTabExecution>
