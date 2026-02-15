import icon from '../../assets/icon.png'
import type { AiTurn } from '../../stores/tabs'
import { Icon } from '../ui/Icon'
import { PromptInput } from './PromptInput'

const DEFAULT_SUGGESTIONS = [
  'Show the top 10 largest tables',
  'Find users who signed up this month',
  'List tables with no indexes',
  'Show recent query activity',
]

interface AiChatPanelProps {
  turns: AiTurn[]
  currentTurnIndex: number
  prompt: string
  error: string | null
  isGenerating: boolean
  suggestions?: string[]
  onPromptChange: (prompt: string) => void
  onSubmit: (prompt: string) => void
  onNavigate: (index: number) => void
}

export function AiChatPanel({
  turns,
  currentTurnIndex,
  prompt,
  error,
  isGenerating,
  suggestions,
  onPromptChange,
  onSubmit,
  onNavigate,
}: AiChatPanelProps) {
  const currentTurn = turns.length > 0 ? turns[currentTurnIndex] : null
  const chips = suggestions?.length ? suggestions : DEFAULT_SUGGESTIONS

  const handleSubmit = () => {
    const trimmed = prompt.trim()
    if (!trimmed || isGenerating) return
    onSubmit(trimmed)
  }

  return (
    <div className="flex h-full flex-col bg-white dark:bg-surface-950">
      {/* Welcome screen — no turns, not loading */}
      {!currentTurn && !isGenerating && (
        <div className="flex flex-1 flex-col items-center justify-center px-6 py-8 animate-fade-in">
          <img src={icon} alt="pglet mascot" className="mb-2 h-20 w-20 opacity-70" />
          <h2 className="mb-1 text-lg font-medium text-gray-700 dark:text-gray-300">
            What would you like to know?
          </h2>
          <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">
            Ask about your data in natural language
          </p>
          <div className="w-full max-w-md">
            <PromptInput
              value={prompt}
              onChange={onPromptChange}
              onSubmit={handleSubmit}
              placeholder="Describe a query, e.g. Show all users who signed up this month"
              submitLabel="Generate"
            />
          </div>
          <div className="mt-4 grid w-full max-w-md grid-cols-2 gap-2">
            {chips.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => onPromptChange(s)}
                className="flex items-center gap-1.5 rounded-full border border-surface-200 px-3 py-1.5 text-xs text-gray-600 transition-colors hover:border-accent-400 hover:text-accent-600 dark:border-surface-700 dark:text-gray-400 dark:hover:border-accent-500 dark:hover:text-accent-400"
              >
                <Icon name="bolt" className="h-2.5 w-2.5 flex-shrink-0 text-accent-500" />
                <span className="truncate">{s}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Loading state */}
      {isGenerating && (
        <div className="flex flex-1 flex-col items-center justify-center animate-fade-in">
          <Icon name="wand" className="mb-3 h-5 w-5 animate-pulse text-accent-500" />
          <div className="flex items-center gap-1.5">
            <span className="typing-dot h-1.5 w-1.5 rounded-full bg-gray-400" />
            <span className="typing-dot h-1.5 w-1.5 rounded-full bg-gray-400" />
            <span className="typing-dot h-1.5 w-1.5 rounded-full bg-gray-400" />
          </div>
          <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
            Generating your query...
          </p>
          <div className="mt-4 w-48 space-y-2">
            <div className="h-2 w-full rounded animate-shimmer-line" />
            <div className="h-2 w-full rounded animate-shimmer-line" style={{ animationDelay: '0.2s' }} />
            <div className="h-2 w-3/4 rounded animate-shimmer-line" style={{ animationDelay: '0.4s' }} />
          </div>
        </div>
      )}

      {/* Iteration result — has turns, not loading */}
      {currentTurn && !isGenerating && (
        <>
          {/* Header bar with version nav + prompt summary */}
          <div className="flex min-w-0 items-center border-b border-surface-200 px-3 py-1.5 dark:border-surface-800">
            <div className="flex items-center gap-1">
              <button
                type="button"
                disabled={currentTurnIndex === 0}
                onClick={() => onNavigate(currentTurnIndex - 1)}
                className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-surface-100 hover:text-gray-600 disabled:opacity-30 dark:hover:bg-surface-800 dark:hover:text-gray-300"
              >
                <Icon name="chevron-left" className="h-2.5 w-2.5" />
              </button>
              <span className="text-[11px] tabular-nums text-gray-500 dark:text-gray-400">
                {currentTurnIndex + 1} / {turns.length}
              </span>
              <button
                type="button"
                disabled={currentTurnIndex === turns.length - 1}
                onClick={() => onNavigate(currentTurnIndex + 1)}
                className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-surface-100 hover:text-gray-600 disabled:opacity-30 dark:hover:bg-surface-800 dark:hover:text-gray-300"
              >
                <Icon name="chevron-right" className="h-2.5 w-2.5" />
              </button>
            </div>
            <span className="ml-2 min-w-0 truncate text-[11px] text-gray-400 dark:text-gray-500">
              {currentTurn.prompt}
            </span>
          </div>

          {/* Content area — key forces remount for fade-in on navigation */}
          <div key={currentTurnIndex} className="min-h-0 flex-1 overflow-y-auto px-3 py-3 animate-fade-in">
            {/* Prompt that produced this result */}
            <div className="mb-3 rounded-md border-l-2 border-l-accent-400 bg-surface-50 px-3 py-2 dark:border-l-accent-600 dark:bg-surface-900">
              <div className="flex items-center gap-1.5">
                <Icon name="wand" className="h-2.5 w-2.5 text-accent-500" />
                <p className="text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500">Prompt</p>
              </div>
              <p className="mt-1 text-sm leading-relaxed text-gray-700 dark:text-gray-300">
                {currentTurn.prompt}
              </p>
            </div>

            {/* Error */}
            {error && (
              <div className="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2.5 dark:border-red-900/40 dark:bg-red-900/20">
                <div className="flex items-start gap-2">
                  <Icon name="xmark" className="mt-0.5 h-3 w-3 flex-shrink-0 text-red-500 dark:text-red-400" />
                  <div>
                    <p className="text-[10px] font-medium uppercase tracking-wider text-red-500 dark:text-red-400">
                      Error
                    </p>
                    <p className="mt-0.5 text-sm leading-relaxed text-red-700 dark:text-red-400">
                      {error}
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Explanation */}
            {currentTurn.explanation && (
              <div className="mb-3 rounded-md border-l-2 border-l-accent-300 bg-surface-50 px-3 py-2 dark:border-l-accent-700 dark:bg-surface-900">
                <div className="flex items-center gap-1.5">
                  <Icon name="lightbulb" className="h-2.5 w-2.5 text-accent-500" />
                  <p className="text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500">Explanation</p>
                </div>
                <p className="mt-1 text-sm leading-relaxed text-gray-600 dark:text-gray-400">
                  {currentTurn.explanation}
                </p>
              </div>
            )}
          </div>

          {/* Refine input — pinned to bottom */}
          <div className="flex-shrink-0 border-t border-surface-200 px-3 py-2 dark:border-surface-800">
            <PromptInput
              value={prompt}
              onChange={onPromptChange}
              onSubmit={handleSubmit}
              placeholder="Refine this query..."
              submitLabel="Refine"
              rows={2}
            />
          </div>
        </>
      )}
    </div>
  )
}
