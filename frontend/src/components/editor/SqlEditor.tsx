import Editor, { type OnMount } from '@monaco-editor/react'
import { useCallback, useRef, useImperativeHandle, forwardRef } from 'react'
import type { editor as MonacoEditor } from 'monaco-editor'
import { formatDialect } from '@avallete/sql-formatter/lite'
import { postgresql } from '@avallete/sql-formatter/languages/postgresql'
import { createCompletionProvider } from './autocomplete'
import { useThemeStore } from '../../stores/theme'

let completionRegistered = false
let themesRegistered = false

export interface SqlEditorHandle {
  getSelectedText: () => string | undefined
  formatSql: () => void
}

interface SqlEditorProps {
  value: string
  onChange: (value: string) => void
  onRun: (selectedText?: string) => void
  onExplain: () => void
  onSave?: () => void
}

export const SqlEditor = forwardRef<SqlEditorHandle, SqlEditorProps>(function SqlEditor(
  { value, onChange, onRun, onExplain, onSave },
  ref,
) {
  const editorRef = useRef<MonacoEditor.IStandaloneCodeEditor | null>(null)

  useImperativeHandle(ref, () => ({
    getSelectedText: () => {
      const ed = editorRef.current
      if (!ed) return undefined
      const selection = ed.getSelection()
      const model = ed.getModel()
      if (selection && !selection.isEmpty() && model) {
        const text = model.getValueInRange(selection)
        return text.trim() ? text : undefined
      }
      return undefined
    },
    formatSql: () => {
      const ed = editorRef.current
      if (!ed) return
      const model = ed.getModel()
      if (!model) return
      const text = model.getValue()
      if (!text.trim()) return
      try {
        const formatted = formatDialect(text, {
          dialect: postgresql,
          tabWidth: 2,
          keywordCase: 'upper',
        })
        model.setValue(formatted)
      } catch {
        // leave text unchanged on parse error
      }
    },
  }))
  const onRunRef = useRef(onRun)
  const onExplainRef = useRef(onExplain)
  const onSaveRef = useRef(onSave)
  onRunRef.current = onRun
  onExplainRef.current = onExplain
  onSaveRef.current = onSave

  const handleMount: OnMount = useCallback(
    (editor, monaco) => {
      editorRef.current = editor

      if (!themesRegistered) {
        monaco.editor.defineTheme('pglet-light', {
          base: 'vs',
          inherit: true,
          rules: [
            { token: 'keyword', foreground: '3d7568', fontStyle: 'bold' },
            { token: 'keyword.sql', foreground: '3d7568', fontStyle: 'bold' },
            { token: 'operator.sql', foreground: '4f9385' },
            { token: 'predefined.sql', foreground: '4f9385' },
            { token: 'string.sql', foreground: 'b45309' },
            { token: 'number', foreground: 'c2410c' },
            { token: 'comment', foreground: 'a8a29e', fontStyle: 'italic' },
          ],
          colors: {
            'editor.selectionBackground': '#b1e0d4',
            'editor.inactiveSelectionBackground': '#d6efe8',
            'editor.selectionHighlightBackground': '#d6efe880',
            'editorCursor.foreground': '#4f9385',
          },
        })
        monaco.editor.defineTheme('pglet-dark', {
          base: 'vs-dark',
          inherit: true,
          rules: [
            { token: 'keyword', foreground: 'b1e0d4', fontStyle: 'bold' },
            { token: 'keyword.sql', foreground: 'b1e0d4', fontStyle: 'bold' },
            { token: 'operator.sql', foreground: '92c5b9' },
            { token: 'predefined.sql', foreground: '92c5b9' },
            { token: 'string.sql', foreground: 'fcd34d' },
            { token: 'number', foreground: 'fb923c' },
            { token: 'comment', foreground: '78716c', fontStyle: 'italic' },
          ],
          colors: {
            'editor.selectionBackground': '#2f5b51',
            'editor.inactiveSelectionBackground': '#2f5b5180',
            'editor.selectionHighlightBackground': '#2f5b5160',
            'editorCursor.foreground': '#b1e0d4',
          },
        })
        themesRegistered = true
      }

      if (!completionRegistered) {
        monaco.languages.registerCompletionItemProvider('sql', createCompletionProvider(monaco))
        completionRegistered = true
      }

      // Use addCommand for Cmd+Enter — more reliable than addAction for keybindings
      editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
        const selection = editor.getSelection()
        const model = editor.getModel()
        if (selection && !selection.isEmpty() && model) {
          const selectedText = model.getValueInRange(selection)
          if (selectedText.trim()) {
            onRunRef.current(selectedText)
            return
          }
        }
        onRunRef.current()
      })

      // Cmd+Shift+Enter: run selection only (no fallback)
      editor.addCommand(
        monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.Enter,
        () => {
          const selection = editor.getSelection()
          const model = editor.getModel()
          if (selection && !selection.isEmpty() && model) {
            const selectedText = model.getValueInRange(selection)
            if (selectedText.trim()) {
              onRunRef.current(selectedText)
            }
          }
        },
      )

      editor.addAction({
        id: 'explain-query',
        label: 'Explain Query',
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
        run: () => onExplainRef.current(),
      })

      editor.addAction({
        id: 'save-query',
        label: 'Save Query',
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
        run: () => onSaveRef.current?.(),
      })

      editor.addAction({
        id: 'format-sql',
        label: 'Format SQL',
        keybindings: [monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyF],
        run: (ed) => {
          const model = ed.getModel()
          if (!model) return
          const text = model.getValue()
          if (!text.trim()) return
          try {
            const formatted = formatDialect(text, {
              dialect: postgresql,
              tabWidth: 2,
              keywordCase: 'upper',
            })
            model.setValue(formatted)
          } catch {
            // leave text unchanged on parse error
          }
        },
      })

      editor.focus()
    },
    [],
  )

  const theme = useThemeStore((s) => s.theme)
  const isDark =
    theme === 'dark' ||
    (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)

  return (
    <Editor
      height="100%"
      defaultLanguage="sql"
      theme={isDark ? 'pglet-dark' : 'pglet-light'}
      value={value}
      onChange={(v) => onChange(v ?? '')}
      onMount={handleMount}
      options={{
        minimap: { enabled: false },
        lineNumbers: 'on',
        fontSize: 13,
        fontFamily: "'IBM Plex Mono', 'JetBrains Mono', monospace",
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        automaticLayout: true,
        tabSize: 2,
        padding: { top: 8 },
        quickSuggestions: true,
        suggestOnTriggerCharacters: true,
      }}
    />
  )
})
