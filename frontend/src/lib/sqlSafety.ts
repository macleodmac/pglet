/**
 * Flags statements worth a confirmation before running: UPDATE/DELETE without
 * a WHERE, DROP, TRUNCATE and ALTER ... DROP. It is a guard rail against
 * slips, not a parser: it only looks at top-level keywords, skipping strings,
 * quoted identifiers, comments and dollar-quoted bodies.
 */

export interface DestructiveStatement {
  /** The statement text, trimmed. */
  sql: string
  /** Why it was flagged, e.g. "DELETE without WHERE affects every row". */
  reason: string
}

interface Word {
  text: string // upper-cased
  depth: number // parenthesis depth
}

interface Statement {
  sql: string
  words: Word[]
}

const WORD_START = /[A-Za-z_]/
const WORD_CHAR = /[A-Za-z0-9_$]/

/** Splits SQL into statements of top-level-aware words. */
function scan(sql: string): Statement[] {
  const statements: Statement[] = []
  let words: Word[] = []
  let depth = 0
  let start = 0
  let i = 0

  const endStatement = (end: number) => {
    const text = sql.slice(start, end).trim()
    if (words.length > 0) statements.push({ sql: text, words })
    words = []
    depth = 0
  }

  while (i < sql.length) {
    const c = sql[i]
    const next = sql[i + 1]

    if (c === '-' && next === '-') {
      const nl = sql.indexOf('\n', i)
      i = nl === -1 ? sql.length : nl + 1
    } else if (c === '/' && next === '*') {
      // Block comments nest in Postgres.
      let nesting = 1
      i += 2
      while (i < sql.length && nesting > 0) {
        if (sql[i] === '/' && sql[i + 1] === '*') {
          nesting++
          i += 2
        } else if (sql[i] === '*' && sql[i + 1] === '/') {
          nesting--
          i += 2
        } else {
          i++
        }
      }
    } else if (c === "'") {
      // E'...' strings allow backslash escapes; '' is an escaped quote in both.
      const escapes = i > 0 && (sql[i - 1] === 'E' || sql[i - 1] === 'e')
      i++
      while (i < sql.length) {
        if (escapes && sql[i] === '\\') {
          i += 2
        } else if (sql[i] === "'" && sql[i + 1] === "'") {
          i += 2
        } else if (sql[i] === "'") {
          i++
          break
        } else {
          i++
        }
      }
    } else if (c === '"') {
      i++
      while (i < sql.length) {
        if (sql[i] === '"' && sql[i + 1] === '"') i += 2
        else if (sql[i] === '"') {
          i++
          break
        } else i++
      }
    } else if (c === '$' && /[A-Za-z_$]/.test(next ?? '') && !(i > 0 && WORD_CHAR.test(sql[i - 1]))) {
      const tag = /^\$([A-Za-z_][A-Za-z0-9_]*)?\$/.exec(sql.slice(i))
      if (tag) {
        const close = sql.indexOf(tag[0], i + tag[0].length)
        i = close === -1 ? sql.length : close + tag[0].length
      } else {
        i++
      }
    } else if (c === '(') {
      depth++
      i++
    } else if (c === ')') {
      depth = Math.max(0, depth - 1)
      i++
    } else if (c === ';') {
      endStatement(i)
      i++
      start = i
    } else if (WORD_START.test(c)) {
      let j = i + 1
      while (j < sql.length && WORD_CHAR.test(sql[j])) j++
      words.push({ text: sql.slice(i, j).toUpperCase(), depth })
      i = j
    } else {
      i++
    }
  }
  endStatement(sql.length)
  return statements
}

const MAIN_VERBS = new Set(['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'MERGE', 'VALUES', 'TABLE'])

function check(stmt: Statement): string | null {
  const top = stmt.words.filter((w) => w.depth === 0)
  if (top.length === 0) return null

  let verbIndex = 0
  if (top[0].text === 'WITH') {
    // Skip the CTE list: its bodies are parenthesised, so the main statement
    // starts at the first top-level verb.
    verbIndex = top.findIndex((w, i) => i > 0 && MAIN_VERBS.has(w.text))
    if (verbIndex === -1) return null
  }
  const verb = top[verbIndex].text
  const rest = top.slice(verbIndex + 1)

  switch (verb) {
    case 'UPDATE':
    case 'DELETE':
      return rest.some((w) => w.text === 'WHERE') ? null : `${verb} without WHERE affects every row`
    case 'TRUNCATE':
      return 'TRUNCATE removes every row'
    case 'DROP': {
      const objectType = rest
        .slice(0, 2)
        .map((w) => w.text)
        .filter((w) => w !== 'IF')
      const kind = objectType[0] === 'MATERIALIZED' || objectType[0] === 'FOREIGN' ? objectType.join(' ') : objectType[0]
      return `DROP ${kind ?? ''}`.trim()
    }
    case 'ALTER':
      return rest.some((w) => w.text === 'DROP') ? 'ALTER ... DROP removes part of an object' : null
    default:
      return null
  }
}

export function findDestructiveStatements(sql: string): DestructiveStatement[] {
  const found: DestructiveStatement[] = []
  for (const stmt of scan(sql)) {
    const reason = check(stmt)
    if (reason) found.push({ sql: stmt.sql, reason })
  }
  return found
}
