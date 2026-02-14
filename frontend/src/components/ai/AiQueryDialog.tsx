import { useCallback, useEffect, useRef, useState } from 'react'

import { useAiGenerate } from '../../api/queries'
import { highlightSql } from './highlightSql'

interface AiQueryDialogProps {
  database: string
  onInsert: (sql: string) => void
  onRun: (sql: string) => void
  onClose: () => void
}

type ChatMessage =
  | { role: 'user'; content: string }
  | { role: 'assistant'; sql: string; explanation: string }
  | { role: 'error'; content: string }

export function AiQueryDialog({ database, onInsert, onRun, onClose }: AiQueryDialogProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [prompt, setPrompt] = useState('')
  const [copied, setCopied] = useState<number | null>(null)
  const generate = useAiGenerate()
  const threadRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      threadRef.current?.scrollTo({ top: threadRef.current.scrollHeight, behavior: 'smooth' })
    })
  }, [])

  useEffect(scrollToBottom, [messages, generate.isPending, scrollToBottom])

  const handleSubmit = () => {
    const trimmed = prompt.trim()
    if (!trimmed || generate.isPending) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    const nextMessages = [...messages, userMsg]
    setMessages(nextMessages)
    setPrompt('')

    // Build history for API: convert prior assistant messages to content strings
    const apiMessages = messages.flatMap((m): Array<{ role: string; content: string }> => {
      if (m.role === 'user') return [{ role: 'user', content: m.content }]
      if (m.role === 'assistant') return [{ role: 'assistant', content: `SQL: ${m.sql}\nExplanation: ${m.explanation}` }]
      return []
    })

    generate.mutate(
      { prompt: trimmed, database, messages: apiMessages.length > 0 ? apiMessages : undefined },
      {
        onSuccess: (data) => {
          setMessages((prev) => [...prev, { role: 'assistant', sql: data.sql, explanation: data.explanation }])

        },
        onError: (err) => {
          setMessages((prev) => [...prev, { role: 'error', content: err.message }])
        },
      },
    )
  }

  const handleCopy = (sql: string, index: number) => {
    navigator.clipboard.writeText(sql)
    setCopied(index)
    setTimeout(() => setCopied(null), 1500)
  }

  const assistantMessages = messages.filter((m) => m.role === 'assistant')
  const lastAssistantIndex = messages.reduce((acc, m, i) => (m.role === 'assistant' ? i : acc), -1)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="flex max-h-[80vh] w-full max-w-2xl animate-slide-up flex-col rounded-xl border border-surface-200 bg-white shadow-lg shadow-black/5 dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/30"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-surface-200 px-4 py-3 dark:border-surface-800">
          <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-200">
            AI SQL Assistant
          </h2>
          <button
            onClick={onClose}
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-surface-100 hover:text-gray-600 dark:hover:bg-surface-800 dark:hover:text-gray-300"
            aria-label="Close"
          >
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" aria-hidden="true">
              <path d="M1 1l12 12M13 1L1 13" />
            </svg>
          </button>
        </div>

        {/* Thread area */}
        <div ref={threadRef} className="flex-1 overflow-y-auto px-4 py-4">
          {messages.length === 0 && !generate.isPending && (
            <div className="flex h-full items-center justify-center">
              <p className="text-center text-xs text-gray-400 dark:text-gray-500">
                Describe what you want to query.
                <br />
                <span className="text-[10px]">Your database schema is automatically included as context.</span>
              </p>
            </div>
          )}

          <div className="space-y-3">
            {messages.map((msg, i) => {
              if (msg.role === 'user') {
                return (
                  <div key={i} className="flex justify-end">
                    <div className="max-w-[80%] rounded-lg bg-accent-50 px-3 py-2 text-xs text-gray-800 dark:bg-accent-700/20 dark:text-gray-200">
                      {msg.content}
                    </div>
                  </div>
                )
              }

              if (msg.role === 'error') {
                return (
                  <div key={i} className="flex justify-start">
                    <div className="max-w-[80%] rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-400">
                      {msg.content}
                    </div>
                  </div>
                )
              }

              // assistant
              const isLatest = i === lastAssistantIndex
              return (
                <div key={i} className="flex justify-start">
                  <div className="w-full max-w-[90%] space-y-2">
                    {/* SQL block */}
                    <div className="group relative rounded-lg border border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-800">
                      <div className="absolute right-1.5 top-1.5 opacity-0 transition-opacity group-hover:opacity-100">
                        <button
                          type="button"
                          onClick={() => handleCopy(msg.sql, i)}
                          className="rounded px-1.5 py-0.5 text-[10px] text-gray-400 hover:bg-surface-200 hover:text-gray-600 dark:hover:bg-surface-700 dark:hover:text-gray-300"
                        >
                          {copied === i ? 'Copied!' : 'Copy'}
                        </button>
                      </div>
                      <pre
                        className="overflow-x-auto p-3 font-mono text-xs leading-relaxed"
                        dangerouslySetInnerHTML={{ __html: highlightSql(msg.sql) }}
                      />
                    </div>

                    {/* Explanation */}
                    {msg.explanation && (
                      <p className="text-xs leading-relaxed text-gray-500 dark:text-gray-400">
                        {msg.explanation}
                      </p>
                    )}

                    {/* Actions */}
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => { onInsert(msg.sql); onClose() }}
                        className={isLatest
                          ? 'rounded border border-gray-300 px-2.5 py-1 text-xs font-medium text-gray-700 hover:bg-surface-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-surface-800'
                          : 'rounded px-2.5 py-1 text-xs text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300'
                        }
                      >
                        Insert
                      </button>
                      <button
                        type="button"
                        onClick={() => { onRun(msg.sql); onClose() }}
                        className={isLatest
                          ? 'rounded bg-accent-500 px-2.5 py-1 text-xs font-medium text-white hover:bg-accent-600'
                          : 'rounded px-2.5 py-1 text-xs text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300'
                        }
                      >
                        Insert & Run
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}

            {/* Typing indicator */}
            {generate.isPending && (
              <div className="flex justify-start">
                <div className="flex gap-1 rounded-lg bg-surface-50 px-3 py-2 dark:bg-surface-800">
                  <span className="typing-dot h-1.5 w-1.5 rounded-full bg-gray-400" />
                  <span className="typing-dot h-1.5 w-1.5 rounded-full bg-gray-400" />
                  <span className="typing-dot h-1.5 w-1.5 rounded-full bg-gray-400" />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Input area */}
        <div className="border-t border-surface-200 px-4 py-3 dark:border-surface-800">
          <div className="flex items-end gap-2">
            <textarea
              ref={textareaRef}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Describe what you want to query..."
              className="block max-h-24 min-h-[2.5rem] flex-1 resize-none rounded-lg border border-gray-300 bg-white px-3 py-2 text-xs focus:border-accent-400 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-200"
              rows={1}
              // eslint-disable-next-line jsx-a11y/no-autofocus
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault()
                  handleSubmit()
                }
              }}
              onInput={(e) => {
                const el = e.currentTarget
                el.style.height = 'auto'
                el.style.height = `${Math.min(el.scrollHeight, 96)}px`
              }}
            />
            <button
              type="button"
              onClick={handleSubmit}
              disabled={generate.isPending || !prompt.trim()}
              className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-accent-500 text-white hover:bg-accent-600 disabled:opacity-40"
              title="Send (Cmd+Enter)"
              aria-label="Send message"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                <path d="M1.5 1.5l13 6.5-13 6.5 2-6.5-2-6.5zm2.4 5.5l-1.2 4 8.6-4-8.6-4 1.2 4h5.1v1H3.9z" />
              </svg>
            </button>
          </div>
          <p className="mt-1.5 text-[10px] text-gray-400 dark:text-gray-500">
            {assistantMessages.length > 0 ? 'Send a follow-up to refine the query' : 'Press Cmd+Enter to send'}
          </p>
        </div>
      </div>
    </div>
  )
}
