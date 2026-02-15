import { useRef } from 'react'
import { Icon } from '../ui/Icon'

interface PromptInputProps {
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  placeholder: string
  submitLabel: string
  rows?: number
  disabled?: boolean
}

export function PromptInput({
  value,
  onChange,
  onSubmit,
  placeholder,
  submitLabel,
  rows = 3,
  disabled,
}: PromptInputProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  return (
    <div className="flex flex-col gap-1.5">
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
            e.preventDefault()
            onSubmit()
          }
        }}
        placeholder={placeholder}
        rows={rows}
        className="w-full resize-none rounded-lg border border-surface-300 bg-white px-3 py-2 text-xs placeholder-gray-400 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500 dark:border-surface-700 dark:bg-surface-900 dark:text-gray-100 dark:placeholder-gray-500"
      />
      <div className="flex items-center justify-between">
        <span className="text-[10px] text-gray-400 dark:text-gray-500">Cmd+Enter to send</span>
        <button
          type="button"
          onClick={onSubmit}
          disabled={disabled || !value.trim()}
          className="flex items-center gap-1.5 rounded bg-accent-500 px-3 py-1 text-[11px] font-medium text-white hover:bg-accent-600 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Icon name="wand" className="h-3 w-3" />
          {submitLabel}
        </button>
      </div>
    </div>
  )
}
