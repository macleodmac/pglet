import { useState, useEffect } from 'react'
import { useSettings, useUpdateSettings } from '../../api/queries'
import { useThemeStore } from '../../stores/theme'

interface SettingsDialogProps {
  onClose: () => void
}

export function SettingsDialog({ onClose }: SettingsDialogProps) {
  const { data: settings } = useSettings()
  const updateSettings = useUpdateSettings()

  const theme = useThemeStore((s) => s.theme)
  const setTheme = useThemeStore((s) => s.setTheme)
  const [apiKey, setApiKey] = useState('')

  useEffect(() => {
    if (settings) {
      setApiKey(settings.ai_api_key || '')
    }
  }, [settings])

  const handleSave = () => {
    const updates: Record<string, string> = {}
    if (apiKey !== (settings?.ai_api_key ?? '')) updates.ai_api_key = apiKey
    if (Object.keys(updates).length > 0) {
      updateSettings.mutate(updates, { onSuccess: onClose })
    } else {
      onClose()
    }
  }

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

        <label className="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">AI API Key (Anthropic)</label>
        <input
          type="password"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          className="mb-3 block w-full rounded border border-gray-300 bg-white px-2.5 py-1.5 text-sm focus:border-accent-400 focus:outline-none focus:ring-1 focus:ring-accent-400 dark:border-gray-600 dark:bg-surface-800 dark:text-gray-200"
          placeholder="sk-ant-..."
        />

        {updateSettings.error && (
          <p className="mb-3 text-xs text-danger">{updateSettings.error.message}</p>
        )}

        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-surface-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-surface-800"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={updateSettings.isPending}
            className="rounded bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50"
          >
            {updateSettings.isPending ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
