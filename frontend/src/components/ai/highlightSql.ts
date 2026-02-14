const KEYWORDS = new Set([
  'SELECT', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER',
  'FULL', 'CROSS', 'ON', 'AS', 'AND', 'OR', 'NOT', 'IN', 'IS', 'NULL',
  'LIKE', 'ILIKE', 'BETWEEN', 'EXISTS', 'HAVING', 'GROUP', 'ORDER', 'BY',
  'LIMIT', 'OFFSET', 'UNION', 'ALL', 'DISTINCT', 'INSERT', 'INTO',
  'VALUES', 'UPDATE', 'SET', 'DELETE', 'CREATE', 'ALTER', 'DROP', 'TABLE',
  'INDEX', 'VIEW', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'ASC', 'DESC',
  'COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'CAST', 'COALESCE', 'NULLIF',
  'TRUE', 'FALSE', 'WITH', 'RECURSIVE', 'RETURNING', 'CONFLICT', 'DO',
  'NOTHING', 'LATERAL', 'FETCH', 'NEXT', 'ROWS', 'ONLY', 'FOR',
])

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

export function highlightSql(sql: string): string {
  // Tokenize: strings, comments, numbers, words, whitespace, other
  const tokens = sql.match(
    /--[^\n]*|'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|\d+(?:\.\d+)?|\b[A-Za-z_]\w*\b|\s+|./g
  )
  if (!tokens) return escapeHtml(sql)

  return tokens
    .map((t) => {
      if (t.startsWith('--')) {
        return `<span class="text-gray-400 dark:text-gray-500 italic">${escapeHtml(t)}</span>`
      }
      if (t.startsWith("'") || t.startsWith('"')) {
        return `<span class="text-amber-600 dark:text-amber-400">${escapeHtml(t)}</span>`
      }
      if (/^\d/.test(t)) {
        return `<span class="text-indigo-500 dark:text-indigo-400">${escapeHtml(t)}</span>`
      }
      if (KEYWORDS.has(t.toUpperCase())) {
        return `<span class="text-accent-600 dark:text-accent-400 font-semibold">${escapeHtml(t)}</span>`
      }
      return escapeHtml(t)
    })
    .join('')
}
