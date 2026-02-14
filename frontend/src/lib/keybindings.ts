export const KEYBINDINGS = {
  runQuery: { key: 'Enter', meta: true, label: 'Cmd+Enter' },
  runSelection: { key: 'Enter', meta: true, shift: true, label: 'Cmd+Shift+Enter' },
  explainQuery: { key: 'e', meta: true, label: 'Cmd+E' },
  saveQuery: { key: 's', meta: true, label: 'Cmd+S' },
  cancelQuery: { key: 'Escape', label: 'Escape' },
  openSavedQueries: { key: 'o', meta: true, label: 'Cmd+O' },
} as const
