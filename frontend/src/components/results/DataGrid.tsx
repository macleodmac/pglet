import { useRef, useState, useCallback } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { CellDetail } from './CellDetail'
import { ContextMenu } from '../shared/ContextMenu'

interface DataGridProps {
  columns: string[]
  columnTypes: string[]
  rows: (string | null)[][]
  rowHeight?: number
  rowOffset?: number
}

export function DataGrid({ columns, columnTypes, rows, rowHeight = 26, rowOffset = 0 }: DataGridProps) {
  const parentRef = useRef<HTMLDivElement>(null)
  const [columnWidths, setColumnWidths] = useState<Record<number, number>>({})
  const [selectedCell, setSelectedCell] = useState<{
    row: number
    col: number
    value: string | null
    type: string
  } | null>(null)
  const [copiedCell, setCopiedCell] = useState<string | null>(null)
  const copiedTimeout = useRef<ReturnType<typeof setTimeout>>(null)
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    value: string | null
    column: string
  } | null>(null)

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan: 20,
  })

  const getWidth = (i: number) => columnWidths[i] ?? 150

  const handleResizeStart = useCallback(
    (colIdx: number, startX: number) => {
      const startWidth = getWidth(colIdx)
      const onMove = (e: MouseEvent) => {
        const delta = e.clientX - startX
        setColumnWidths((prev) => ({
          ...prev,
          [colIdx]: Math.max(50, startWidth + delta),
        }))
      }
      const onUp = () => {
        document.removeEventListener('mousemove', onMove)
        document.removeEventListener('mouseup', onUp)
      }
      document.addEventListener('mousemove', onMove)
      document.addEventListener('mouseup', onUp)
    },
    [columnWidths],
  )

  const handleAutoFit = useCallback(
    (colIdx: number) => {
      const canvas = document.createElement('canvas')
      const ctx = canvas.getContext('2d')
      if (!ctx) return
      // Match the font used in cells: monospace 12px (text-xs)
      ctx.font = '12px monospace'
      const padding = 20 // px-2 = 8px each side + 4px buffer
      // Measure header (bold)
      ctx.font = 'bold 12px monospace'
      let max = ctx.measureText(columns[colIdx]).width + padding
      // Measure all row values
      ctx.font = '12px monospace'
      for (const row of rows) {
        const cell = row[colIdx]
        const text = cell === null ? 'NULL' : cell
        const w = ctx.measureText(text).width + padding
        if (w > max) max = w
      }
      setColumnWidths((prev) => ({
        ...prev,
        [colIdx]: Math.max(50, Math.min(Math.ceil(max), 600)),
      }))
    },
    [columns, rows],
  )

  const totalWidth = columns.reduce((sum, _, i) => sum + getWidth(i), 0) + 36 // +36 for row number col

  if (columns.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-400">
        No results
      </div>
    )
  }

  return (
    <>
      <div ref={parentRef} className="h-full overflow-auto">
        <div style={{ minWidth: totalWidth }}>
          {/* Header */}
          <div className="sticky top-0 z-10 flex border-b border-surface-200 bg-surface-50 dark:border-surface-800 dark:bg-surface-900">
            <div className="flex w-[36px] flex-shrink-0 items-center justify-center border-r border-surface-200 font-mono text-xs text-gray-400 dark:border-surface-800" style={{ height: rowHeight }}>
              #
            </div>
            {columns.map((col, i) => (
              <div
                key={i}
                className="relative flex flex-shrink-0 items-center border-r border-surface-200 px-2 font-mono text-xs font-semibold text-gray-600 dark:border-surface-800 dark:text-gray-300"
                style={{ width: getWidth(i), height: rowHeight }}
              >
                <span className="truncate">{col}</span>
                <div
                  className="absolute right-0 top-0 h-full w-1 cursor-col-resize hover:bg-accent-400"
                  onMouseDown={(e) => handleResizeStart(i, e.clientX)}
                  onDoubleClick={() => handleAutoFit(i)}
                />
              </div>
            ))}
          </div>

          {/* Rows */}
          <div
            style={{
              height: rowVirtualizer.getTotalSize(),
              position: 'relative',
            }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const row = rows[virtualRow.index]
              return (
                <div
                  key={virtualRow.index}
                  className="absolute left-0 flex border-b border-surface-200/50 hover:bg-accent-50/50 dark:border-surface-800/50 dark:hover:bg-accent-900/10"
                  style={{
                    top: virtualRow.start,
                    height: virtualRow.size,
                  }}
                >
                  <div className="flex h-full w-[36px] flex-shrink-0 items-center justify-center border-r border-surface-200/50 font-mono text-xs text-gray-400 dark:border-surface-800/50">
                    {rowOffset + virtualRow.index + 1}
                  </div>
                  {row.map((cell, colIdx) => (
                    <div
                      key={colIdx}
                      className="group/cell relative flex h-full flex-shrink-0 cursor-pointer items-center border-r border-surface-200/50 px-2 font-mono text-xs dark:border-surface-800/50"
                      style={{ width: getWidth(colIdx) }}
                      onClick={() =>
                        setSelectedCell({
                          row: virtualRow.index,
                          col: colIdx,
                          value: cell,
                          type: columnTypes[colIdx],
                        })
                      }
                      onContextMenu={(e) => {
                        e.preventDefault()
                        setContextMenu({
                          x: e.clientX,
                          y: e.clientY,
                          value: cell,
                          column: columns[colIdx],
                        })
                      }}
                    >
                      {cell === null ? (
                        <span className="italic text-gray-300 dark:text-gray-600">NULL</span>
                      ) : (
                        <span className="truncate text-gray-800 dark:text-gray-200">{cell}</span>
                      )}
                      <button
                        type="button"
                        className={`absolute right-1 top-1/2 hidden -translate-y-1/2 rounded p-0.5 group-hover/cell:block ${
                          copiedCell === `${virtualRow.index}-${colIdx}`
                            ? 'text-success'
                            : 'text-gray-400 hover:bg-surface-200 hover:text-gray-600 dark:hover:bg-surface-700 dark:hover:text-gray-300'
                        }`}
                        onClick={(e) => {
                          e.stopPropagation()
                          const key = `${virtualRow.index}-${colIdx}`
                          navigator.clipboard.writeText(cell ?? '').then(() => {
                            setCopiedCell(key)
                            if (copiedTimeout.current) clearTimeout(copiedTimeout.current)
                            copiedTimeout.current = setTimeout(() => setCopiedCell(null), 1500)
                          })
                        }}
                        title="Copy"
                      >
                        <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                        </svg>
                      </button>
                    </div>
                  ))}
                </div>
              )
            })}
          </div>
        </div>
      </div>

      {selectedCell && (
        <CellDetail
          column={columns[selectedCell.col]}
          columnType={columnTypes[selectedCell.col]}
          value={selectedCell.value}
          onClose={() => setSelectedCell(null)}
        />
      )}

      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={[
            {
              label: 'Copy value',
              onClick: () => navigator.clipboard.writeText(contextMenu.value ?? ''),
            },
            {
              label: 'Copy as JSON',
              onClick: () =>
                navigator.clipboard.writeText(JSON.stringify(contextMenu.value)),
            },
            { label: '', separator: true, onClick: () => {} },
            {
              label: `Filter by ${contextMenu.column}`,
              onClick: () => {
                /* TODO: wire to query filter */
              },
            },
          ]}
          onClose={() => setContextMenu(null)}
        />
      )}
    </>
  )
}
