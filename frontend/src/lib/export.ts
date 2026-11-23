import type { QueryResult } from '../api/generated'
import { apiFetch } from '../api/client'

export type ExportFormat = 'csv' | 'json'

function csvField(v: string | null): string {
  if (v === null) return ''
  return /[",\r\n]/.test(v) ? `"${v.replace(/"/g, '""')}"` : v
}

function toCsv(result: QueryResult): string {
  const lines = [result.columns.map(csvField).join(',')]
  for (const row of result.rows) lines.push(row.map(csvField).join(','))
  return lines.join('\n') + '\n'
}

/** JSON text for one value, built by hand so int8, numeric and big JSON numbers keep every digit. */
function jsonField(v: string | null, type: string): string {
  if (v === null) return 'null'
  switch (type) {
    case 'INT2':
    case 'INT4':
    case 'FLOAT4':
    case 'FLOAT8':
      if (/^-?\d+(\.\d+)?(e[-+]?\d+)?$/i.test(v)) return v
      break
    case 'BOOL':
      return v === 'true' ? 'true' : 'false'
    case 'JSON':
    case 'JSONB':
      return v
  }
  return JSON.stringify(v)
}

function toJson(result: QueryResult): string {
  const keys = result.columns.map((c) => JSON.stringify(c))
  const rows = result.rows.map(
    (row) => '{' + row.map((v, i) => `${keys[i]}:${jsonField(v, result.column_types[i])}`).join(',') + '}',
  )
  return '[' + rows.join(',\n') + ']\n'
}

function save(blob: Blob, format: ExportFormat) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `export.${format}`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

/** Download exactly the rows on screen; nothing is re-run. */
export function exportResult(result: QueryResult, format: ExportFormat) {
  const text = format === 'csv' ? toCsv(result) : toJson(result)
  save(new Blob([text], { type: format === 'csv' ? 'text/csv' : 'application/json' }), format)
}

/**
 * Download every row of a single read-only statement. The server re-runs it
 * in a read-only transaction, so a statement that writes fails instead.
 */
export async function exportAllRows(sql: string, format: ExportFormat) {
  const res = await apiFetch('/api/export', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query: sql, format }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || 'Export failed')
  }
  save(await res.blob(), format)
}
