import type { AiTurn } from '../../stores/tabs'
import { Icon } from '../ui/Icon'
import { PromptInput } from './PromptInput'
import icon from '../../assets/icon.png'

interface AiChatPanelProps {
  turns: AiTurn[]
  currentTurnIndex: number
  prompt: string
  error: string | null
  isGenerating: boolean
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
  onPromptChange,
  onSubmit,
  onNavigate,
}: AiChatPanelProps) {
  const currentTurn = turns.length > 0 ? turns[currentTurnIndex] : null

  const handleSubmit = () => {
    const trimmed = prompt.trim()
    if (!trimmed || isGenerating) return
    onSubmit(trimmed)
  }

  return (
    <div className="flex h-full flex-col bg-white dark:bg-surface-950">
      {/* Welcome screen — no turns, not loading */}
      {!currentTurn && !isGenerating && (
        <div className="flex flex-1 flex-col items-center justify-center px-6 py-8">
          <img src={icon} alt="pglet mascot" className="mb-2 h-20 w-20 opacity-70" />
          <h2 className="mb-6 text-lg font-medium text-gray-700 dark:text-gray-300">
            What would you like to know?
          </h2>
          <div className="w-full max-w-md">
            <PromptInput
              value={prompt}
              onChange={onPromptChange}
              onSubmit={handleSubmit}
              placeholder="Describe a query, e.g. Show all users who signed up this month"
              submitLabel="Generate"
            />
          </div>
        </div>
      )}

      {/* Loading state */}
      {isGenerating && (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            <span className="typing-dot" />
            <span className="typing-dot" style={{ animationDelay: '0.2s' }} />
            <span className="typing-dot" style={{ animationDelay: '0.4s' }} />
            <span className="ml-1">Generating...</span>
          </div>
        </div>
      )}

      {/* Iteration result — has turns, not loading */}
      {currentTurn && !isGenerating && (
        <>
          {/* Header bar with version nav */}
          <div className="flex items-center border-b border-surface-200 px-3 py-1.5 dark:border-surface-800">
            <div className="flex items-center gap-1">
              <button
                type="button"
                disabled={currentTurnIndex === 0}
                onClick={() => onNavigate(currentTurnIndex - 1)}
                className="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-surface-100 hover:text-gray-600 disabled:opacity-30 dark:hover:bg-surface-800 dark:hover:text-gray-300"
              >
                <Icon name="chevron-left" className="h-2 w-2" />
              </button>
              <span className="text-[11px] tabular-nums text-gray-500 dark:text-gray-400">
                {currentTurnIndex + 1} / {turns.length}
              </span>
              <button
                type="button"
                disabled={currentTurnIndex === turns.length - 1}
                onClick={() => onNavigate(currentTurnIndex + 1)}
                className="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-surface-100 hover:text-gray-600 disabled:opacity-30 dark:hover:bg-surface-800 dark:hover:text-gray-300"
              >
                <Icon name="chevron-right" className="h-2 w-2" />
              </button>
            </div>
          </div>

          {/* Content area */}
          <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
            {/* Prompt that produced this result */}
            <div className="mb-2 rounded-md bg-surface-50 px-3 py-2 dark:bg-surface-900">
              <p className="text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500">Prompt</p>
              <p className="mt-0.5 text-sm leading-relaxed text-gray-700 dark:text-gray-300">
                {currentTurn.prompt}
              </p>
            </div>

            {/* Error */}
            {error && (
              <div className="mb-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
                {error}
              </div>
            )}

            {/* Explanation */}
            {currentTurn.explanation && (
              <div className="mb-3 rounded-md bg-surface-50 px-3 py-2 dark:bg-surface-900">
                <p className="text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500">Explanation</p>
                <p className="mt-0.5 text-sm leading-relaxed text-gray-600 dark:text-gray-400">
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
