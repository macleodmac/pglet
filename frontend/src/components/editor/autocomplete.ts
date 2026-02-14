import type * as monaco from 'monaco-editor'
import { queryClient } from '../../queryClient'
import type { SchemaGroup, SchemaObject } from '../../api/generated'

const PG_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'IS', 'NULL', 'AS',
  'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'FULL', 'CROSS', 'ON',
  'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE',
  'CREATE', 'ALTER', 'DROP', 'TABLE', 'INDEX', 'VIEW', 'SCHEMA',
  'GROUP', 'BY', 'ORDER', 'ASC', 'DESC', 'LIMIT', 'OFFSET',
  'HAVING', 'DISTINCT', 'UNION', 'ALL', 'INTERSECT', 'EXCEPT',
  'EXISTS', 'BETWEEN', 'LIKE', 'ILIKE', 'SIMILAR', 'TO',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'CAST', 'COALESCE', 'NULLIF', 'GREATEST', 'LEAST',
  'COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'ARRAY_AGG', 'STRING_AGG',
  'NOW', 'CURRENT_TIMESTAMP', 'CURRENT_DATE', 'CURRENT_TIME',
  'TRUE', 'FALSE', 'DEFAULT',
  'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK', 'CONSTRAINT',
  'WITH', 'RECURSIVE', 'RETURNING',
  'EXPLAIN', 'ANALYZE', 'VERBOSE',
  'BEGIN', 'COMMIT', 'ROLLBACK', 'SAVEPOINT',
  'GRANT', 'REVOKE',
  'TRIGGER', 'FUNCTION', 'PROCEDURE', 'RETURNS', 'LANGUAGE',
  'IF', 'ELSIF', 'LOOP', 'FOR', 'WHILE', 'DECLARE',
  'TEXT', 'INTEGER', 'BIGINT', 'SMALLINT', 'BOOLEAN', 'VARCHAR', 'CHAR',
  'NUMERIC', 'DECIMAL', 'REAL', 'DOUBLE', 'PRECISION', 'SERIAL', 'BIGSERIAL',
  'TIMESTAMP', 'DATE', 'TIME', 'INTERVAL', 'JSON', 'JSONB', 'UUID', 'BYTEA',
]

const PG_FUNCTIONS = [
  'count', 'sum', 'avg', 'min', 'max', 'array_agg', 'string_agg', 'json_agg', 'jsonb_agg',
  'now', 'current_timestamp', 'current_date', 'age', 'date_trunc', 'date_part', 'extract',
  'length', 'lower', 'upper', 'trim', 'ltrim', 'rtrim', 'substring', 'replace', 'concat',
  'split_part', 'regexp_replace', 'regexp_matches', 'left', 'right', 'repeat', 'reverse',
  'abs', 'ceil', 'floor', 'round', 'trunc', 'mod', 'power', 'sqrt', 'random',
  'coalesce', 'nullif', 'greatest', 'least',
  'row_number', 'rank', 'dense_rank', 'lag', 'lead', 'first_value', 'last_value', 'ntile',
  'to_char', 'to_date', 'to_timestamp', 'to_number',
  'json_build_object', 'jsonb_build_object', 'json_extract_path', 'jsonb_extract_path_text',
  'array_length', 'unnest', 'generate_series',
  'pg_size_pretty', 'pg_total_relation_size', 'pg_table_size', 'pg_indexes_size',
]

// Parse the SQL text to find table references after FROM and JOIN
function findTableReferences(text: string): string[] {
  const tables: string[] = []
  const regex = /\b(?:FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)/gi
  let match
  while ((match = regex.exec(text)) !== null) {
    tables.push(match[1])
  }
  return tables
}

export function createCompletionProvider(monacoInstance: typeof monaco): monaco.languages.CompletionItemProvider {
  return {
    triggerCharacters: ['.', ' '],
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position)
      const range: monaco.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      }

      const suggestions: monaco.languages.CompletionItem[] = []
      const objects = queryClient.getQueryData<Record<string, SchemaGroup>>(['objects'])

      // SQL keywords
      for (const kw of PG_KEYWORDS) {
        suggestions.push({
          label: kw,
          kind: monacoInstance.languages.CompletionItemKind.Keyword,
          insertText: kw,
          range,
          sortText: '3_' + kw,
        })
      }

      // Built-in functions
      for (const fn of PG_FUNCTIONS) {
        suggestions.push({
          label: fn,
          kind: monacoInstance.languages.CompletionItemKind.Function,
          insertText: fn + '($0)',
          insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range,
          sortText: '2_' + fn,
        })
      }

      if (!objects) return { suggestions }

      // Table and view names from schema objects
      const allTables: SchemaObject[] = []
      for (const [schema, group] of Object.entries(objects)) {
        const schemaObjects = [
          ...group.tables,
          ...group.views,
          ...group.materialized_views,
        ]
        for (const obj of schemaObjects) {
          allTables.push(obj)
          const label = schema === 'public' ? obj.name : `${schema}.${obj.name}`
          suggestions.push({
            label,
            kind: monacoInstance.languages.CompletionItemKind.Class,
            insertText: label,
            detail: obj.type,
            range,
            sortText: '1_' + label,
          })
        }

        // User-defined functions
        for (const fn of group.functions) {
          const label = schema === 'public' ? fn.name : `${schema}.${fn.name}`
          suggestions.push({
            label,
            kind: monacoInstance.languages.CompletionItemKind.Function,
            insertText: label + '($0)',
            insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
            sortText: '2_' + label,
          })
        }
      }

      // Column completion: check if cursor is after a table reference
      const fullText = model.getValue()
      const tableRefs = findTableReferences(fullText)

      for (const ref of tableRefs) {
        const qualifiedName = ref.includes('.') ? ref : `public.${ref}`
        const [schema, table] = qualifiedName.split('.')

        // Look up cached column data
        const columns = queryClient.getQueryData<Array<{ name: string; type: string }>>(
          ['table-columns', ref],
        ) ?? queryClient.getQueryData<Array<{ name: string; type: string }>>(
          ['table-columns', `${schema}.${table}`],
        )

        if (columns) {
          for (const col of columns) {
            suggestions.push({
              label: col.name,
              kind: monacoInstance.languages.CompletionItemKind.Field,
              insertText: col.name,
              detail: `${col.type} (${ref})`,
              range,
              sortText: '0_' + col.name,
            })
          }
        }
      }

      return { suggestions }
    },
  }
}
