import { useSettings } from '../../api/queries'
import { useThemeStore } from '../../stores/theme'

interface SettingsDialogProps {
  onClose: () => void
}

export function SettingsDialog({ onClose }: SettingsDialogProps) {
  const { data: settings } = useSettings()

  const theme = useThemeStore((s) => s.theme)
  const setTheme = useThemeStore((s) => s.setTheme)

  const aiKeySet = settings?.ai_api_key_set === 'true'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="w-full max-w-md animate-slide-up rounded-xl border border-surface-200 bg-white p-5 shadow-lg shadow-black/5 dark:border-surface-800 dark:bg-surface-900 dark:shadow-black/30"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-sm font-semibold text-gray-800 dark:text-gray-200">Settings</h2>

        <p className="mb-1.5 text-xs font-medium text-gray-600 dark:text-gray-400">Theme</p>
        <div className="mb-4 flex gap-1">
          {(['light', 'dark', 'system'] as const).map((t) => (
            <button
              type="button"
              key={t}
              onClick={() => setTheme(t)}
              className={`rounded px-3 py-1.5 text-xs font-medium capitalize ${
                theme === t
                  ? 'bg-accent-500 text-white'
                  : 'border border-gray-300 text-gray-700 hover:bg-surface-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-surface-800'
              }`}
            >
              {t}
            </button>
          ))}
        </div>

        <p className="mb-1.5 text-xs font-medium text-gray-600 dark:text-gray-400">AI API Key (Anthropic)</p>
        <p className={`mb-4 text-xs ${aiKeySet ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-500'}`}>
          {aiKeySet ? 'ANTHROPIC_API_KEY is set' : 'ANTHROPIC_API_KEY not set — export it in your shell to enable AI features'}
        </p>

        <div className="flex justify-end">
          <button
            onClick={onClose}
            className="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-surface-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-surface-800"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
