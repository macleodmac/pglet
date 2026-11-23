import { useCallback } from 'react'
import type { Tab } from '../../stores/tabs'
import { useTabStore } from '../../stores/tabs'
import { useExecutionStore, useTabRun, type TabRun } from '../../stores/execution'
import { useSettingsStore } from '../../stores/settings'
import { useAiFix, autoNameTab } from '../../api/queries'
import { queryClient } from '../../api/queryClient'
import {
  runQuery,
  explainQuery,
  cancelQuery,
  beginTransaction,
  commitTransaction,
  rollbackTransaction,
} from '../../api/client'
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

function errorMessage(error: unknown): string {
  return (error as { error?: string })?.error || 'Request failed'
}

/** Name an unnamed tab after its first successful query, unless the user names it first. */
function maybeAutoName(tabId: string, sql: string) {
  const current = () => useTabStore.getState().tabs.find((t) => t.id === tabId)
  const tab = current()
  if (!tab || tab.userRenamed || tab.aiRenamed || tab.savedQueryId) return
  const { updateTab } = useTabStore.getState()
  updateTab(tabId, { aiRenamed: true })
  autoNameTab(sql).then((name) => {
    const latest = current()
    if (name && latest && !latest.userRenamed && !latest.savedQueryId) updateTab(tabId, { title: name })
  })
}

/**
 * Execution for one tab. State lives in the execution store keyed by tab ID,
 * and requests write their results there by the ID they were started with,
 * so a response that arrives after the user switched tabs lands in its own tab.
 */
export function useTabExecution(tab: Tab) {
  const run = useTabRun(tab.id)
  const safeWrites = useSettingsStore((s) => s.safeWrites)
  const appendAiTurn = useTabStore((s) => s.appendAiTurn)
  const updateTab = useTabStore((s) => s.updateTab)
  const aiFix = useAiFix()
  const tabId = tab.id

  const patch = useCallback(
    (updates: Partial<TabRun>) => useExecutionStore.getState().patch(tabId, updates),
    [tabId],
  )

  const syncTransaction = useCallback(
    (inTransaction: boolean | undefined) => {
      useTabStore.getState().updateTab(tabId, { inTransaction: !!inTransaction })
      if (!inTransaction) patch({ heldWrite: null })
    },
    [tabId, patch],
  )

  const isBusy = () => {
    const r = useExecutionStore.getState().runs[tabId]
    return !!(r?.isRunning || r?.txPending)
  }

  const execute = useCallback(
    async (query: string, useSafeWrites: boolean) => {
      if (!query.trim() || isBusy()) return
      patch({ isRunning: true, resultSql: query, txError: null })
      aiFix.reset()
      const res = await runQuery({ body: { query, tab_id: tabId, safe_writes: useSafeWrites } })
      if (!res.data) {
        patch({ isRunning: false, result: errorResult(errorMessage(res.error)) })
        return
      }
      const data = res.data
      patch({ isRunning: false, result: data, ...(data.held_for_review ? { heldWrite: data } : {}) })
      // A query can open or abort a transaction, e.g. an error inside one.
      syncTransaction(data.in_transaction)
      queryClient.invalidateQueries({ queryKey: ['history'] })
      if (!data.error) maybeAutoName(tabId, query)
    },
    [tabId, aiFix, patch, syncTransaction],
  )

  const runSql = useCallback((sql?: string) => execute(sql || tab.sql, safeWrites), [execute, tab.sql, safeWrites])

  /** Re-run the last query without safe writes, for statements that can't run in a transaction. */
  const runUnguarded = useCallback(() => {
    const sql = useExecutionStore.getState().runs[tabId]?.resultSql
    if (sql) execute(sql, false)
  }, [execute, tabId])

  const explain = useCallback(
    async (sql?: string) => {
      const query = sql || tab.sql
      if (!query.trim() || isBusy()) return
      patch({ isRunning: true, resultSql: query, txError: null })
      aiFix.reset()
      const res = await explainQuery({ body: { query, tab_id: tabId } })
      patch({ isRunning: false, result: res.data ?? errorResult(errorMessage(res.error)) })
    },
    [tab.sql, tabId, aiFix, patch],
  )

  const cancel = useCallback(() => {
    cancelQuery({ body: { tab_id: tabId } })
  }, [tabId])

  const txAction = useCallback(
    async (action: typeof beginTransaction) => {
      patch({ txError: null, txPending: true })
      const res = await action({ body: { tab_id: tabId } })
      patch({ txPending: false })
      if (res.data) {
        syncTransaction(res.data.in_transaction)
        queryClient.invalidateQueries({ queryKey: ['history'] })
      } else {
        patch({ txError: errorMessage(res.error) })
        // The server may have dropped the transaction anyway (e.g. an aborted commit).
        await useTabStore.getState().syncOpenTransactions()
      }
    },
    [tabId, patch, syncTransaction],
  )

  const requestFix = useCallback(() => {
    if (!run.result?.error) return
    aiFix.mutate({ sql: run.resultSql, error: run.result.error })
  }, [run.result, run.resultSql, aiFix])

  const applyFix = useCallback(
    (andRun: boolean) => {
      const fix = aiFix.data
      if (!fix) return
      if (tab.type === 'ai') {
        const firstLine = run.result?.error?.split('\n')[0].replace(/^ERROR:\s+/, '') ?? ''
        appendAiTurn(tab.id, { prompt: `Fix: ${firstLine}`, sql: fix.sql, explanation: fix.explanation })
      } else {
        // The failed SQL may have been a selection, so replace only that part.
        const sql = tab.sql.includes(run.resultSql) ? tab.sql.replace(run.resultSql, () => fix.sql) : fix.sql
        updateTab(tab.id, { sql })
      }
      aiFix.reset()
      if (andRun) runSql(fix.sql)
    },
    [aiFix, tab.type, tab.id, tab.sql, run.result, run.resultSql, appendAiTurn, updateTab, runSql],
  )

  return {
    result: run.result,
    resultSql: run.resultSql,
    isRunning: run.isRunning,
    txPending: run.txPending,
    txError: run.txError,
    run: runSql,
    runUnguarded,
    explain,
    cancel,
    begin: () => txAction(beginTransaction),
    commit: () => txAction(commitTransaction),
    rollback: () => txAction(rollbackTransaction),
    heldWrite: tab.inTransaction ? run.heldWrite : null,
    fix: aiFix,
    requestFix,
    applyFix,
  }
}

export type TabExecution = ReturnType<typeof useTabExecution>
