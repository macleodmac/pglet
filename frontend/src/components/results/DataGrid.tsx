import { useRef, useState, useCallback } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { CellDetail } from './CellDetail'
import { ContextMenu } from '../shared/ContextMenu'

interface DataGridProps {
  columns: string[]
  columnTypes: string[]
  rows: (string | null)[][]
}

export function DataGrid({ columns, columnTypes, rows }: DataGridProps) {
  const parentRef = useRef<HTMLDivElement>(null)
  const [columnWidths, setColumnWidths] = useState<Record<number, number>>({})
  const [selectedCell, setSelectedCell] = useState<{
    row: number
    col: number
    value: string | null
    type: string
  } | null>(null)
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    value: string | null
    column: string
  } | null>(null)

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 32,
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

  const totalWidth = columns.reduce((sum, _, i) => sum + getWidth(i), 0) + 50 // +50 for row number col

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
            <div className="flex h-8 w-[50px] flex-shrink-0 items-center justify-center border-r border-surface-200 font-mono text-xs text-gray-400 dark:border-surface-800">
              #
            </div>
            {columns.map((col, i) => (
              <div
                key={i}
                className="relative flex h-8 flex-shrink-0 items-center border-r border-surface-200 px-2 font-mono text-xs font-semibold text-gray-600 dark:border-surface-800 dark:text-gray-300"
                style={{ width: getWidth(i) }}
              >
                <span className="truncate">{col}</span>
                <div
                  className="absolute right-0 top-0 h-full w-1 cursor-col-resize hover:bg-accent-400"
                  onMouseDown={(e) => handleResizeStart(i, e.clientX)}
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
                  <div className="flex h-full w-[50px] flex-shrink-0 items-center justify-center border-r border-surface-200/50 font-mono text-xs text-gray-400 dark:border-surface-800/50">
                    {virtualRow.index + 1}
                  </div>
                  {row.map((cell, colIdx) => (
                    <div
                      key={colIdx}
                      className="flex h-full flex-shrink-0 cursor-pointer items-center border-r border-surface-200/50 px-2 font-mono text-xs dark:border-surface-800/50"
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
