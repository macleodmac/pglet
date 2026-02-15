import { useState, useRef, useCallback, useLayoutEffect } from 'react'
import type { Tab } from '../../stores/tabs'
import { useTabStore } from '../../stores/tabs'
import { useTableRows, useTableColumns, useTableIndexes, useTableConstraints, useTableInfo } from '../../api/queries'
import { DataGrid } from './DataGrid'
import { Pagination } from './Pagination'

const BASE_ROW_HEIGHT = 26
const MIN_PAGE_SIZE = 10

export function TableBrowser({ tableName, tab }: { tableName: string; tab: Tab }) {
  const updateTab = useTabStore((s) => s.updateTab)
  const activeView = tab.activeView || 'rows'

  const contentRef = useRef<HTMLDivElement>(null)
  const [pageSize, setPageSize] = useState(MIN_PAGE_SIZE)
  const [rowHeight, setRowHeight] = useState(BASE_ROW_HEIGHT)
  const [page, setPage] = useState(0)
  const [sortColumn, _setSortColumn] = useState<string>()
  const [sortOrder, _setSortOrder] = useState<'ASC' | 'DESC'>()

  const calcPageSize = useCallback(() => {
    if (!contentRef.current) return
    const h = contentRef.current.clientHeight
    // +1 accounts for the header row which also uses rowHeight
    const exactRows = h / BASE_ROW_HEIGHT - 1
    // Round down to nearest multiple of 5
    const rounded = Math.max(MIN_PAGE_SIZE, Math.floor(exactRows / 5) * 5)
    setPageSize(rounded)
    // Divide full height by (rows + header) — rows grow slightly to fill the space
    setRowHeight(Math.floor(h / (rounded + 1)))
  }, [])

  useLayoutEffect(() => {
    calcPageSize()
    const el = contentRef.current
    if (!el) return
    const ro = new ResizeObserver(calcPageSize)
    ro.observe(el)
    return () => ro.disconnect()
  }, [calcPageSize])

  const rowsQuery = useTableRows(tableName, {
    limit: pageSize,
    offset: page * pageSize,
    sort_column: sortColumn,
    sort_order: sortOrder,
  })

  const columnsQuery = useTableColumns(tableName)
  const indexesQuery = useTableIndexes(tableName)
  const constraintsQuery = useTableConstraints(tableName)
  const infoQuery = useTableInfo(tableName)

  const views = [
    { key: 'rows' as const, label: 'Rows' },
    { key: 'structure' as const, label: 'Structure' },
    { key: 'indexes' as const, label: 'Indexes' },
    { key: 'constraints' as const, label: 'Constraints' },
  ]

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Info bar */}
      <div className="flex items-center gap-3 border-b border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900">
        <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{tableName}</span>
        {infoQuery.data && (
          <span className="text-xs text-gray-400">
            {infoQuery.data.row_estimate >= 0 && <>~{infoQuery.data.row_estimate.toLocaleString()} rows · </>}{infoQuery.data.total_size}
          </span>
        )}
      </div>

      {/* View tabs */}
      <div className="flex border-b border-surface-200 dark:border-surface-800">
        {views.map((v) => (
          <button
            type="button"
            key={v.key}
            onClick={() => updateTab(tab.id, { activeView: v.key })}
            className={`px-4 py-1.5 text-xs font-medium ${
              activeView === v.key
                ? 'border-b-2 border-accent-500 text-accent-600 dark:text-accent-400'
                : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
            }`}
          >
            {v.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div ref={contentRef} className="min-h-0 flex-1 overflow-auto">
        {activeView === 'rows' && rowsQuery.data && (
          <DataGrid
            columns={rowsQuery.data.columns}
            columnTypes={rowsQuery.data.column_types || rowsQuery.data.columns.map(() => '')}
            rows={rowsQuery.data.rows}
            rowHeight={rowHeight}
            rowOffset={page * pageSize}
          />
        )}

        {activeView === 'structure' && columnsQuery.data && (
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-900">
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">#</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Name</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Type</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Nullable</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Default</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">PK</th>
              </tr>
            </thead>
            <tbody>
              {columnsQuery.data.map((col) => (
                <tr key={col.name} className="border-b border-surface-200/50 dark:border-surface-800/50">
                  <td className="px-3 py-1.5 text-gray-400">{col.position}</td>
                  <td className="px-3 py-1.5 font-medium text-gray-800 dark:text-gray-200">{col.name}</td>
                  <td className="px-3 py-1.5 text-gray-600 dark:text-gray-400">{col.type}</td>
                  <td className="px-3 py-1.5 text-gray-500">{col.nullable ? 'YES' : 'NO'}</td>
                  <td className="px-3 py-1.5 text-gray-500">{col.default_value ?? ''}</td>
                  <td className="px-3 py-1.5">{col.is_primary_key ? '🔑' : ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {activeView === 'indexes' && indexesQuery.data && (
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-900">
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Name</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Unique</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Primary</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Definition</th>
              </tr>
            </thead>
            <tbody>
              {indexesQuery.data.map((idx) => (
                <tr key={idx.name} className="border-b border-surface-200/50 dark:border-surface-800/50">
                  <td className="px-3 py-1.5 font-medium text-gray-800 dark:text-gray-200">{idx.name}</td>
                  <td className="px-3 py-1.5 text-gray-500">{idx.is_unique ? 'YES' : 'NO'}</td>
                  <td className="px-3 py-1.5 text-gray-500">{idx.is_primary ? 'YES' : 'NO'}</td>
                  <td className="px-3 py-1.5 text-gray-500">{idx.definition}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {activeView === 'constraints' && constraintsQuery.data && (
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-900">
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Name</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Type</th>
                <th className="px-3 py-2 text-left font-semibold text-gray-600 dark:text-gray-300">Definition</th>
              </tr>
            </thead>
            <tbody>
              {constraintsQuery.data.map((con) => (
                <tr key={con.name} className="border-b border-surface-200/50 dark:border-surface-800/50">
                  <td className="px-3 py-1.5 font-medium text-gray-800 dark:text-gray-200">{con.name}</td>
                  <td className="px-3 py-1.5 text-gray-500">{con.type}</td>
                  <td className="px-3 py-1.5 text-gray-500">{con.definition}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {activeView === 'rows' && rowsQuery.data && (
        <Pagination
          page={page}
          pageSize={pageSize}
          totalCount={rowsQuery.data.total_count}
          onPageChange={setPage}
        />
      )}
    </div>
  )
}
