import { useMemo, useState } from 'react'
import { Icon } from '../ui/Icon'

interface PlanNode {
  text: string
  detail: string[]
  children: PlanNode[]
  depth: number
}

function parsePlanLines(lines: string[]): PlanNode[] {
  const roots: PlanNode[] = []
  const stack: { node: PlanNode; indent: number }[] = []

  for (const line of lines) {
    // Find the arrow "->  " or top-level node (no arrow, zero indent)
    const arrowMatch = line.match(/^(\s*)->\s+(.*)$/)
    const isTopLevel = !arrowMatch && stack.length === 0 && line.trim().length > 0

    if (arrowMatch || isTopLevel) {
      const indent = arrowMatch ? arrowMatch[1].length : 0
      const text = arrowMatch ? arrowMatch[2] : line.trim()

      const node: PlanNode = { text, detail: [], children: [], depth: indent }

      // Pop stack until we find a parent with less indent
      while (stack.length > 0 && stack[stack.length - 1].indent >= indent) {
        stack.pop()
      }

      if (stack.length > 0) {
        stack[stack.length - 1].node.children.push(node)
      } else {
        roots.push(node)
      }

      stack.push({ node, indent })
    } else if (stack.length > 0 && line.trim().length > 0) {
      // Detail line (indented text without arrow, belongs to last node)
      stack[stack.length - 1].node.detail.push(line.trim())
    }
  }

  return roots
}

function PlanNodeRow({ node, level = 0 }: { node: PlanNode; level?: number }) {
  const [expanded, setExpanded] = useState(true)
  const hasChildren = node.children.length > 0

  // Parse cost/rows/width from text like "Sort  (cost=18.92..18.92 rows=1 width=56)"
  const costMatch = node.text.match(/\(cost=([0-9.]+)\.\.([0-9.]+)\s+rows=(\d+)\s+width=(\d+)\)/)

  return (
    <div>
      <div
        className="group flex items-start gap-1 py-0.5 hover:bg-surface-100 dark:hover:bg-surface-800/50"
        style={{ paddingLeft: `${level * 20 + 8}px` }}
      >
        {hasChildren ? (
          <button
            type="button"
            className="mt-1 flex h-4 w-4 flex-shrink-0 items-center justify-center rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            onClick={() => setExpanded((e) => !e)}
          >
            <Icon name={expanded ? 'chevron-down' : 'chevron-right'} className="h-2 w-2" />
          </button>
        ) : (
          <span className="mt-1 h-4 w-4 flex-shrink-0" />
        )}

        <div className="min-w-0">
          <span className="font-mono text-xs">
            <span className="font-semibold text-gray-800 dark:text-gray-200">
              {costMatch ? node.text.slice(0, node.text.indexOf('(')) : node.text}
            </span>
            {costMatch && (
              <span className="text-gray-500 dark:text-gray-400">
                {' '}cost={costMatch[1]}..{costMatch[2]}
                <span className="text-accent-600 dark:text-accent-400"> rows={costMatch[3]}</span>
                {' '}width={costMatch[4]}
              </span>
            )}
          </span>
          {node.detail.length > 0 && (
            <div className="mt-0.5 space-y-0 pl-1">
              {node.detail.map((d, i) => (
                <div key={i} className="font-mono text-[11px] text-gray-500 dark:text-gray-400">
                  {d}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {expanded &&
        node.children.map((child, i) => (
          <PlanNodeRow key={i} node={child} level={level + 1} />
        ))}
    </div>
  )
}

interface ExplainPlanProps {
  rows: (string | null)[][]
  durationMs: number
}

export function ExplainPlan({ rows, durationMs }: ExplainPlanProps) {
  const lines = useMemo(() => rows.map((r) => r[0] ?? ''), [rows])
  const tree = useMemo(() => parsePlanLines(lines), [lines])
  const rawText = lines.join('\n')

  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(rawText)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-shrink-0 items-center gap-2 border-b border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900">
        <span className="font-mono text-[11px] font-semibold text-gray-600 dark:text-gray-300">
          QUERY PLAN
        </span>
        <span className="font-mono text-[11px] text-gray-400 dark:text-gray-500">
          {rows.length} nodes · {durationMs}ms
        </span>
        <button
          type="button"
          onClick={handleCopy}
          className="ml-auto flex items-center gap-1 rounded px-2 py-0.5 text-[11px] text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
        >
          <Icon name="copy" className="h-2 w-2" />
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>

      <div className="min-h-0 flex-1 overflow-auto py-1">
        {tree.length > 0 ? (
          tree.map((node, i) => <PlanNodeRow key={i} node={node} />)
        ) : (
          <pre className="whitespace-pre-wrap p-3 font-mono text-xs text-gray-800 dark:text-gray-200">
            {rawText}
          </pre>
        )}
      </div>
    </div>
  )
}

/** Detect if a QueryResult is an EXPLAIN output (single "QUERY PLAN" column). */
export function isExplainResult(columns: string[]): boolean {
  return columns.length === 1 && columns[0] === 'QUERY PLAN'
}
