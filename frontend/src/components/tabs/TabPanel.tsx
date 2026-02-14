import type { Tab } from '../../stores/tabs'
import type { SqlEditorHandle } from '../editor/SqlEditor'
import { DataGrid } from '../results/DataGrid'
import { ExplainPlan, isExplainResult } from '../results/ExplainPlan'
import { TableBrowser } from '../results/TableBrowser'
import { SaveQueryDialog } from '../saved-queries/SaveQueryDialog'
import { QueryResultsEmpty } from '../results/QueryResultsEmpty'
import { AiChatPanel } from '../ai/AiChatPanel'

import { useTabStore } from '../../stores/tabs'
import { useConnectionStore } from '../../stores/connection'
import { useRunQuery, useExplainQuery, useCancelQuery, useAiGenerate } from '../../api/queries'
import { lazy, Suspense, useState, useCallback, useEffect, useRef } from 'react'

const SqlEditor = lazy(() => import('../editor/SqlEditor').then((m) => ({ default: m.SqlEditor })))
const FunctionBrowser = lazy(() => import('../results/FunctionBrowser').then((m) => ({ default: m.FunctionBrowser })))
import { Group, Panel, Separator, useDefaultLayout } from 'react-resizable-panels'
import type { QueryResult } from '../../api/generated'
import { Icon } from '../ui/Icon'

async function downloadExport(query: string, format: 'csv' | 'json') {
  const res = await fetch('/api/export', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, format }),
  })
  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error || 'Export failed')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `export.${format}`
  document.body.appendChild(a)
  a.click()
  URL.revokeObjectURL(url)
  a.remove()
}

export function TabPanel({ tab }: { tab: Tab }) {
  const updateTab = useTabStore((s) => s.updateTab)
  const appendAiTurn = useTabStore((s) => s.appendAiTurn)
  const setAiTurnIndex = useTabStore((s) => s.setAiTurnIndex)
  const editorRef = useRef<SqlEditorHandle>(null)
  const runQuery = useRunQuery()
  const explainQuery = useExplainQuery()
  const cancelQuery = useCancelQuery()
  const aiGenerate = useAiGenerate()
  const database = useConnectionStore((s) => s.info?.database ?? '')
  const [result, setResult] = useState<QueryResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)
  const [showSaveDialog, setShowSaveDialog] = useState(false)

  const handleRun = useCallback(
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
          },
          onError: (err) => {
            setResult({
              columns: [],
              column_types: [],
              rows: [],
              row_count: 0,
              duration_ms: 0,
              error: err.message,
            })
            setIsRunning(false)
          },
        },
      )
    },
    [tab.sql, tab.id, runQuery],
  )

  const handleExplain = useCallback(() => {
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
          setResult({
            columns: [],
            column_types: [],
            rows: [],
            row_count: 0,
            duration_ms: 0,
            error: err.message,
          })
          setIsRunning(false)
        },
      },
    )
  }, [tab.sql, tab.id, explainQuery])

  const handleCancel = useCallback(() => {
    cancelQuery.mutate(tab.id)
  }, [tab.id, cancelQuery])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isRunning) {
        handleCancel()
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [isRunning, handleCancel])

  const editorResultsLayout = useDefaultLayout({ id: 'pglet-editor-results' })
  const editorAiLayout = useDefaultLayout({ id: 'pglet-editor-ai' })

  const handleAiSubmit = useCallback(
    (prompt: string) => {
      if (!prompt.trim() || aiGenerate.isPending) return

      updateTab(tab.id, { aiPrompt: '', aiError: null })

      const turns = tab.aiTurns ?? []
      const apiMessages = turns.flatMap((t) => [
        { role: 'user', content: t.prompt },
        { role: 'assistant', content: `SQL: ${t.sql}\nExplanation: ${t.explanation}` },
      ])

      aiGenerate.mutate(
        { prompt, database, messages: apiMessages.length > 0 ? apiMessages : undefined },
        {
          onSuccess: (data) => {
            appendAiTurn(tab.id, { prompt, sql: data.sql, explanation: data.explanation })
          },
          onError: (err) => {
            updateTab(tab.id, { aiError: err.message })
          },
        },
      )
    },
    [tab.id, tab.aiTurns, database, aiGenerate, updateTab, appendAiTurn],
  )

  if (tab.type === 'function' && tab.functionName) {
    return (
      <Suspense>
        <FunctionBrowser functionName={tab.functionName} />
      </Suspense>
    )
  }

  if (tab.type === 'table' && tab.tableName) {
    return <TableBrowser tableName={tab.tableName} tab={tab} />
  }

  const aiHasTurns = (tab.aiTurns?.length ?? 0) > 0
  const aiChatProps = {
    turns: tab.aiTurns ?? [],
    currentTurnIndex: tab.aiCurrentTurnIndex ?? 0,
    prompt: tab.aiPrompt ?? '',
    error: tab.aiError ?? null,
    isGenerating: aiGenerate.isPending,
    onPromptChange: (aiPrompt: string) => updateTab(tab.id, { aiPrompt }),
    onSubmit: handleAiSubmit,
    onNavigate: (index: number) => setAiTurnIndex(tab.id, index),
  }

  if (tab.type === 'ai') {
    return (
      <div className="relative flex flex-1 flex-col overflow-hidden">
        {/* Full-screen welcome overlay — shown before first query */}
        <div
          className={`absolute inset-0 z-10 transition-opacity duration-300 ${
            aiHasTurns ? 'pointer-events-none opacity-0' : 'opacity-100'
          }`}
        >
          <AiChatPanel {...aiChatProps} />
        </div>

        <Group
          orientation="vertical"
          className="min-h-0 flex-1"
          defaultLayout={editorResultsLayout.defaultLayout}
          onLayoutChanged={editorResultsLayout.onLayoutChanged}
        >
          <Panel id="editor" defaultSize="40%" minSize="15%" maxSize="85%">
            <Group
              orientation="horizontal"
              className="h-full"
              defaultLayout={editorAiLayout.defaultLayout}
              onLayoutChanged={editorAiLayout.onLayoutChanged}
            >
              <Panel id="editor-content" defaultSize="50%" minSize="30%" maxSize="70%">
                <Suspense>
                  <SqlEditor
                    ref={editorRef}
                    value={tab.sql}
                    onChange={(sql) => updateTab(tab.id, { sql })}
                    onRun={handleRun}
                    onExplain={handleExplain}
                    onSave={() => setShowSaveDialog(true)}
                  />
                </Suspense>
              </Panel>

              <Separator className="w-px bg-surface-200 transition-colors data-[separator=hover]:bg-accent-500 data-[separator=active]:bg-accent-500 dark:bg-surface-800" />

              <Panel id="ai-chat" minSize="30%">
                <AiChatPanel {...aiChatProps} />
              </Panel>
            </Group>
          </Panel>

          <Separator className="h-px bg-surface-200 transition-colors data-[separator=hover]:bg-accent-500 data-[separator=active]:bg-accent-500 dark:bg-surface-800" />

          <Panel id="results" minSize="10%">
            <div className="flex h-full flex-col">
              <div className="flex flex-shrink-0 items-center gap-1.5 border-y border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900">
                {/* Left: primary query actions */}
                {isRunning ? (
                  <button
                    type="button"
                    onClick={handleCancel}
                    className="flex items-center gap-1 rounded bg-danger px-3 py-1 text-xs font-medium text-white hover:bg-danger/90"
                  >
                    <Icon name="xmark" className="h-3 w-3" />
                    Cancel
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => handleRun(editorRef.current?.getSelectedText())}
                    className="flex items-center gap-1 rounded bg-accent-500 px-3 py-1 text-xs font-medium text-white hover:bg-accent-600"
                  >
                    <Icon name="play" className="h-3 w-3" />
                    Run
                  </button>
                )}
                <button
                  type="button"
                  onClick={handleExplain}
                  disabled={isRunning}
                  className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs font-medium text-gray-700 hover:bg-surface-100 disabled:opacity-50 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
                >
                  <Icon name="lightbulb" className="h-3 w-3" />
                  Explain
                </button>

                {/* Center: result stats */}
                {result && !result.error && !isExplainResult(result.columns) && (
                  <span className="ml-2 font-mono text-[11px] text-gray-400 dark:text-gray-500">
                    {result.row_count} rows · {result.duration_ms}ms
                  </span>
                )}

                {/* Right: tools & export */}
                <div className="ml-auto flex items-center gap-1.5">
                  {result && !result.error && !isExplainResult(result.columns) && (
                    <>
                      <button
                        type="button"
                        onClick={() => downloadExport(tab.sql, 'csv')}
                        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                        title="Export as CSV"
                      >
                        <Icon name="file-csv" className="h-3 w-3" />
                        CSV
                      </button>
                      <button
                        type="button"
                        onClick={() => downloadExport(tab.sql, 'json')}
                        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                        title="Export as JSON"
                      >
                        <Icon name="file-code" className="h-3 w-3" />
                        JSON
                      </button>
                      <span className="mx-0.5 h-3.5 w-px bg-surface-300 dark:bg-surface-700" />
                    </>
                  )}
                  <button
                    type="button"
                    onClick={() => editorRef.current?.formatSql()}
                    className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
                    title="Format SQL (Alt+Shift+F)"
                  >
                    <Icon name="wand" className="h-3 w-3" />
                    Format
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowSaveDialog(true)}
                    className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
                    title="Save query (Cmd+S)"
                  >
                    <Icon name="save" className="h-3 w-3" />
                    Save
                  </button>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-auto">
                {result?.error ? (
                  <div className="p-3 text-sm text-danger dark:text-danger">{result.error}</div>
                ) : result && isExplainResult(result.columns) ? (
                  <ExplainPlan rows={result.rows} durationMs={result.duration_ms} />
                ) : result ? (
                  <DataGrid columns={result.columns} columnTypes={result.column_types} rows={result.rows} />
                ) : (
                  <QueryResultsEmpty />
                )}
              </div>
            </div>
          </Panel>
        </Group>

        {showSaveDialog && (
          <SaveQueryDialog
            sql={tab.sql}
            database={database}
            existingQuery={tab.savedQueryId ? { id: tab.savedQueryId, title: tab.title, sql: tab.sql, description: '', database, tags: '', shared: false, created_at: '', updated_at: '' } : undefined}
            onClose={() => setShowSaveDialog(false)}
            onSaved={() => {
              if (!tab.savedQueryId) {
                // Will be updated when we have the saved query ID from the response
              }
            }}
          />
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <Group
        orientation="vertical"
        className="min-h-0 flex-1"
        defaultLayout={editorResultsLayout.defaultLayout}
        onLayoutChanged={editorResultsLayout.onLayoutChanged}
      >
        <Panel id="editor" defaultSize="40%" minSize="15%" maxSize="85%">
          <Suspense>
            <SqlEditor
              ref={editorRef}
              value={tab.sql}
              onChange={(sql) => updateTab(tab.id, { sql })}
              onRun={handleRun}
              onExplain={handleExplain}
              onSave={() => setShowSaveDialog(true)}
            />
          </Suspense>
        </Panel>

        <Separator className="h-px bg-surface-200 transition-colors data-[separator=hover]:bg-accent-500 data-[separator=active]:bg-accent-500 dark:bg-surface-800" />

        <Panel id="results" minSize="10%">
          <div className="flex h-full flex-col">
            <div className="flex flex-shrink-0 items-center gap-1.5 border-y border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900">
              {/* Left: primary query actions */}
              {isRunning ? (
                <button
                  type="button"
                  onClick={handleCancel}
                  className="flex items-center gap-1 rounded bg-danger px-3 py-1 text-xs font-medium text-white hover:bg-danger/90"
                >
                  <Icon name="xmark" className="h-3 w-3" />
                  Cancel
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => handleRun(editorRef.current?.getSelectedText())}
                  className="flex items-center gap-1 rounded bg-accent-500 px-3 py-1 text-xs font-medium text-white hover:bg-accent-600"
                >
                  <Icon name="play" className="h-3 w-3" />
                  Run
                </button>
              )}
              <button
                type="button"
                onClick={handleExplain}
                disabled={isRunning}
                className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs font-medium text-gray-700 hover:bg-surface-100 disabled:opacity-50 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
              >
                <Icon name="lightbulb" className="h-3 w-3" />
                Explain
              </button>

              {/* Center: result stats */}
              {result && !result.error && !isExplainResult(result.columns) && (
                <span className="ml-2 font-mono text-[11px] text-gray-400 dark:text-gray-500">
                  {result.row_count} rows · {result.duration_ms}ms
                </span>
              )}

              {/* Right: tools & export */}
              <div className="ml-auto flex items-center gap-1.5">
                {result && !result.error && !isExplainResult(result.columns) && (
                  <>
                    <button
                      type="button"
                      onClick={() => downloadExport(tab.sql, 'csv')}
                      className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                      title="Export as CSV"
                    >
                      <Icon name="file-csv" className="h-3 w-3" />
                      CSV
                    </button>
                    <button
                      type="button"
                      onClick={() => downloadExport(tab.sql, 'json')}
                      className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
                      title="Export as JSON"
                    >
                      <Icon name="file-code" className="h-3 w-3" />
                      JSON
                    </button>
                    <span className="mx-0.5 h-3.5 w-px bg-surface-300 dark:bg-surface-700" />
                  </>
                )}
                <button
                  type="button"
                  onClick={() => editorRef.current?.formatSql()}
                  className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
                  title="Format SQL (Alt+Shift+F)"
                >
                  <Icon name="wand" className="h-3 w-3" />
                  Format
                </button>
                <button
                  type="button"
                  onClick={() => setShowSaveDialog(true)}
                  className="flex items-center gap-1 rounded border border-surface-300 px-2 py-1 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
                  title="Save query (Cmd+S)"
                >
                  <Icon name="save" className="h-3 w-3" />
                  Save
                </button>
              </div>
            </div>

            <div className="min-h-0 flex-1 overflow-auto">
              {result?.error ? (
                <div className="p-3 text-sm text-danger dark:text-danger">{result.error}</div>
              ) : result && isExplainResult(result.columns) ? (
                <ExplainPlan rows={result.rows} durationMs={result.duration_ms} />
              ) : result ? (
                <DataGrid columns={result.columns} columnTypes={result.column_types} rows={result.rows} />
              ) : (
                <QueryResultsEmpty />
              )}
            </div>
          </div>
        </Panel>
      </Group>

      {showSaveDialog && (
        <SaveQueryDialog
          sql={tab.sql}
          database={database}
          existingQuery={tab.savedQueryId ? { id: tab.savedQueryId, title: tab.title, sql: tab.sql, description: '', database, tags: '', shared: false, created_at: '', updated_at: '' } : undefined}
          onClose={() => setShowSaveDialog(false)}
          onSaved={() => {
            if (!tab.savedQueryId) {
              // Will be updated when we have the saved query ID from the response
            }
          }}
        />
      )}

    </div>
  )
}
