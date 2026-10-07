import { useState } from 'react'
import type { Column, TableFilter } from '../../api/generated'
import { Icon } from '../ui/Icon'

type FilterOp = TableFilter['op']

const OPS: { op: FilterOp; label: string }[] = [
  { op: 'eq', label: '=' },
  { op: 'neq', label: '≠' },
  { op: 'lt', label: '<' },
  { op: 'lte', label: '≤' },
  { op: 'gt', label: '>' },
  { op: 'gte', label: '≥' },
  { op: 'like', label: 'like' },
  { op: 'ilike', label: 'ilike' },
  { op: 'is_null', label: 'is null' },
  { op: 'not_null', label: 'is not null' },
]

const opLabel = (op: FilterOp) => OPS.find((o) => o.op === op)?.label ?? op
const takesValue = (op: FilterOp) => op !== 'is_null' && op !== 'not_null'

const inputClass =
  'rounded border border-surface-300 bg-white px-1.5 py-0.5 font-mono text-xs text-gray-700 focus:border-accent-400 focus:outline-none dark:border-surface-700 dark:bg-surface-800 dark:text-gray-300'

/** Column filters for the table browser, combined with AND. */
export function FilterBar({
  columns,
  filters,
  onChange,
}: {
  columns: Column[]
  filters: TableFilter[]
  onChange: (filters: TableFilter[]) => void
}) {
  const [column, setColumn] = useState('')
  const [op, setOp] = useState<FilterOp>('eq')
  const [value, setValue] = useState('')

  const selectedColumn = column || columns[0]?.name || ''

  const add = () => {
    if (!selectedColumn) return
    onChange([...filters, takesValue(op) ? { column: selectedColumn, op, value } : { column: selectedColumn, op }])
    setValue('')
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5 border-b border-surface-200 px-3 py-1.5 dark:border-surface-800">
      {filters.map((f, i) => (
        <span
          key={i}
          className="flex items-center gap-1 rounded bg-accent-500/10 py-0.5 pl-2 pr-1 font-mono text-[11px] text-accent-700 dark:text-accent-300"
        >
          {f.column} {opLabel(f.op)}
          {takesValue(f.op) && <span className="text-gray-700 dark:text-gray-200">'{f.value}'</span>}
          <button
            type="button"
            onClick={() => onChange(filters.filter((_, j) => j !== i))}
            className="rounded p-0.5 hover:bg-accent-500/20"
            aria-label={`Remove filter on ${f.column}`}
          >
            <Icon name="xmark" className="h-2 w-2" />
          </button>
        </span>
      ))}

      <form
        className="flex items-center gap-1"
        onSubmit={(e) => {
          e.preventDefault()
          add()
        }}
      >
        <select value={selectedColumn} onChange={(e) => setColumn(e.target.value)} className={inputClass} aria-label="Filter column">
          {columns.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name}
            </option>
          ))}
        </select>
        <select value={op} onChange={(e) => setOp(e.target.value as FilterOp)} className={inputClass} aria-label="Filter operator">
          {OPS.map((o) => (
            <option key={o.op} value={o.op}>
              {o.label}
            </option>
          ))}
        </select>
        {takesValue(op) && (
          <input
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder={op === 'like' || op === 'ilike' ? 'pattern, e.g. %foo%' : 'value'}
            className={`${inputClass} w-40`}
            aria-label="Filter value"
          />
        )}
        <button
          type="submit"
          className="rounded border border-surface-300 px-2 py-0.5 text-xs text-gray-700 hover:bg-surface-100 dark:border-surface-700 dark:text-gray-300 dark:hover:bg-surface-800"
        >
          Add filter
        </button>
      </form>

      {filters.length > 0 && (
        <button
          type="button"
          onClick={() => onChange([])}
          className="ml-auto text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        >
          Clear filters
        </button>
      )}
    </div>
  )
}
