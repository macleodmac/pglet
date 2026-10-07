import type { Tab } from '../../stores/tabs'
import type { SqlEditorHandle } from '../editor/SqlEditor'
import { TableBrowser } from '../browser/TableBrowser'
import { SaveQueryDialog } from '../saved-queries/SaveQueryDialog'
import { AiChatPanel } from '../ai/AiChatPanel'
import { ResultsPanel } from './ResultsPanel'
import { useTabExecution } from './useTabExecution'

import { useTabStore } from '../../stores/tabs'
import { useConnectionStore } from '../../stores/connection'
import { useAiGenerate, useAiSuggestions } from '../../api/queries'
import { lazy, Suspense, useState, useCallback, useRef } from 'react'
import { Group, Panel, Separator, useDefaultLayout } from 'react-resizable-panels'

const SqlEditor = lazy(() => import('../editor/SqlEditor').then((m) => ({ default: m.SqlEditor })))
const FunctionBrowser = lazy(() => import('../browser/FunctionBrowser').then((m) => ({ default: m.FunctionBrowser })))

const separatorClass =
  'bg-surface-200 transition-colors data-[separator=hover]:bg-accent-500 data-[separator=active]:bg-accent-500 dark:bg-surface-800'

export function TabPanel({ tab }: { tab: Tab }) {
  const updateTab = useTabStore((s) => s.updateTab)
  const appendAiTurn = useTabStore((s) => s.appendAiTurn)
  const setAiTurnIndex = useTabStore((s) => s.setAiTurnIndex)
  const editorRef = useRef<SqlEditorHandle>(null)
  const execution = useTabExecution(tab)
  const aiGenerate = useAiGenerate()
  const aiSuggestions = useAiSuggestions()
  const database = useConnectionStore((s) => s.info?.database ?? '')
  const [showSaveDialog, setShowSaveDialog] = useState(false)

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

  const isAi = tab.type === 'ai'
  const aiHasTurns = (tab.aiTurns?.length ?? 0) > 0
  const aiChatProps = {
    turns: tab.aiTurns ?? [],
    currentTurnIndex: tab.aiCurrentTurnIndex ?? 0,
    prompt: tab.aiPrompt ?? '',
    error: tab.aiError ?? null,
    isGenerating: aiGenerate.isPending,
    suggestions: aiSuggestions.data?.suggestions,
    onPromptChange: (aiPrompt: string) => updateTab(tab.id, { aiPrompt }),
    onSubmit: handleAiSubmit,
    onNavigate: (index: number) => setAiTurnIndex(tab.id, index),
  }

  const editor = (
    <Suspense>
      <SqlEditor
        ref={editorRef}
        value={tab.sql}
        onChange={(sql) => updateTab(tab.id, { sql })}
        onRun={execution.run}
        onExplain={execution.explain}
        onAnalyze={execution.analyze}
        onSave={() => setShowSaveDialog(true)}
      />
    </Suspense>
  )

  return (
    <div className={`${isAi ? 'relative ' : ''}flex flex-1 flex-col overflow-hidden`}>
      {isAi && (
        <div
          className={`absolute inset-0 z-20 overflow-hidden transition-opacity duration-300 ${
            aiHasTurns ? 'pointer-events-none opacity-0' : 'opacity-100'
          }`}
        >
          <AiChatPanel {...aiChatProps} />
        </div>
      )}

      <Group
        orientation="vertical"
        className="min-h-0 flex-1"
        defaultLayout={editorResultsLayout.defaultLayout}
        onLayoutChanged={editorResultsLayout.onLayoutChanged}
      >
        <Panel id="editor" defaultSize="40%" minSize="15%" maxSize="85%">
          {isAi ? (
            <Group
              orientation="horizontal"
              className="h-full"
              defaultLayout={editorAiLayout.defaultLayout}
              onLayoutChanged={editorAiLayout.onLayoutChanged}
            >
              <Panel id="editor-content" defaultSize="50%" minSize="30%" maxSize="70%">
                {editor}
              </Panel>

              <Separator className={`w-px ${separatorClass}`} />

              <Panel id="ai-chat" minSize="30%">
                <AiChatPanel {...aiChatProps} />
              </Panel>
            </Group>
          ) : (
            editor
          )}
        </Panel>

        <Separator className={`h-px ${separatorClass}`} />

        <Panel id="results" minSize="10%">
          <ResultsPanel
            sql={tab.sql}
            inTransaction={!!tab.inTransaction}
            execution={execution}
            editorRef={editorRef}
            onSave={() => setShowSaveDialog(true)}
          />
        </Panel>
      </Group>

      {showSaveDialog && (
        <SaveQueryDialog
          sql={tab.sql}
          database={database}
          existingQuery={tab.savedQueryId ? { id: tab.savedQueryId, title: tab.title, sql: tab.sql, description: '', database, tags: '', shared: false, created_at: '', updated_at: '' } : undefined}
          onClose={() => setShowSaveDialog(false)}
        />
      )}
    </div>
  )
}
