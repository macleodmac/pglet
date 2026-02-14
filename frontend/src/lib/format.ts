export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString()
}

export function isJsonString(s: string): boolean {
  if (!s) return false
  const c = s.charAt(0)
  if (c !== '{' && c !== '[') return false
  try {
    JSON.parse(s)
    return true
  } catch {
    return false
  }
}
