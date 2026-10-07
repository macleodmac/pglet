import { useMemo, useState } from 'react'
import { Icon } from '../ui/Icon'
import { HOT_SHARE, parsePlanJson, planStats, type ParsedPlan, type PlanNode as JsonNode } from '../../lib/explainPlan'

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

/** Renders EXPLAIN output: a visual tree for FORMAT JSON, an indented tree for text. */
export function ExplainPlan({ rows, durationMs }: ExplainPlanProps) {
  const lines = useMemo(() => rows.map((r) => r[0] ?? ''), [rows])
  const json = useMemo(() => (lines.length === 1 ? parsePlanJson(lines[0]) : null), [lines])
  const rawText = lines.join('\n')

  return json ? (
    <JsonPlan plan={json} rawText={rawText} />
  ) : (
    <TextPlan lines={lines} rawText={rawText} durationMs={durationMs} />
  )
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  const handleCopy = () => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }
  return (
    <button
      type="button"
      onClick={handleCopy}
      className="flex items-center gap-1 rounded px-2 py-0.5 text-[11px] text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
    >
      <Icon name="copy" className="h-2 w-2" />
      {copied ? 'Copied' : 'Copy'}
    </button>
  )
}

const headerClass =
  'flex flex-shrink-0 items-center gap-2 border-b border-surface-200 bg-surface-50 px-3 py-1.5 dark:border-surface-800 dark:bg-surface-900'

function TextPlan({ lines, rawText, durationMs }: { lines: string[]; rawText: string; durationMs: number }) {
  const tree = useMemo(() => parsePlanLines(lines), [lines])

  return (
    <div className="flex h-full flex-col">
      <div className={headerClass}>
        <span className="font-mono text-[11px] font-semibold text-gray-600 dark:text-gray-300">
          QUERY PLAN
        </span>
        <span className="font-mono text-[11px] text-gray-400 dark:text-gray-500">
          {lines.length} lines · {durationMs}ms
        </span>
        <span className="ml-auto">
          <CopyButton text={rawText} />
        </span>
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

// Node | self time or cost | rows | loops | buffers
const GRID = 'minmax(0, 1fr) 10rem 8.5rem 3.5rem 7.5rem'

function formatMs(ms: number) {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`
  if (ms >= 10) return `${ms.toFixed(0)} ms`
  return `${ms.toFixed(2)} ms`
}

function JsonPlan({ plan, rawText }: { plan: ParsedPlan; rawText: string }) {
  const [showRaw, setShowRaw] = useState(false)
  const stats = useMemo(() => planStats(plan.root), [plan])
  // Shares are of the root's inclusive time (or cost) so they add up to 100%.
  const total = plan.analyzed ? (plan.root.totalMs ?? 0) : plan.totalCost

  return (
    <div className="flex h-full flex-col">
      <div className={headerClass}>
        <span className="font-mono text-[11px] font-semibold text-gray-600 dark:text-gray-300">
          QUERY PLAN
        </span>
        {plan.analyzed ? (
          <span
            className="rounded bg-accent-500/15 px-1.5 py-px text-[10px] font-medium uppercase tracking-wide text-accent-700 dark:text-accent-300"
            title="The statement ran for real timings, then was rolled back"
          >
            Analyzed · rolled back
          </span>
        ) : (
          <span
            className="rounded bg-surface-200 px-1.5 py-px text-[10px] font-medium uppercase tracking-wide text-gray-600 dark:bg-surface-800 dark:text-gray-400"
            title="Planner estimates only; the statement did not run. Use Analyze for actual timings."
          >
            Estimate
          </span>
        )}
        <span className="font-mono text-[11px] text-gray-400 dark:text-gray-500">
          {plan.analyzed && plan.executionMs !== undefined && <>Execution {formatMs(plan.executionMs)} · </>}
          {plan.planningMs !== undefined && <>Planning {formatMs(plan.planningMs)} · </>}
          {!plan.analyzed && <>Cost {plan.totalCost.toLocaleString()} · </>}
          {stats.nodes} nodes
        </span>
        {stats.warnings > 0 && (
          <span className="font-mono text-[11px] text-amber-600 dark:text-amber-400">
            {stats.warnings} {stats.warnings === 1 ? 'warning' : 'warnings'}
          </span>
        )}
        <span className="ml-auto flex items-center gap-1">
          <button
            type="button"
            onClick={() => setShowRaw((r) => !r)}
            className="rounded px-2 py-0.5 text-[11px] text-gray-500 hover:bg-surface-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-surface-800 dark:hover:text-gray-300"
          >
            {showRaw ? 'Tree' : 'JSON'}
          </button>
          <CopyButton text={rawText} />
        </span>
      </div>

      {showRaw ? (
        <pre className="min-h-0 flex-1 overflow-auto p-3 font-mono text-xs text-gray-800 dark:text-gray-200">{rawText}</pre>
      ) : (
        <div className="min-h-0 flex-1 overflow-auto">
          <div
            className="sticky top-0 z-10 grid border-b border-surface-200 bg-surface-50 px-2 py-1 font-mono text-[10px] uppercase tracking-wide text-gray-400 dark:border-surface-800 dark:bg-surface-900 dark:text-gray-500"
            style={{ gridTemplateColumns: GRID }}
          >
            <span>Node</span>
            <span>{plan.analyzed ? 'Self time' : 'Self cost'}</span>
            <span className="text-right">{plan.analyzed ? 'Rows act / est' : 'Est rows'}</span>
            <span className="text-right">{plan.analyzed ? 'Loops' : ''}</span>
            <span className="text-right">{plan.root.buffers ? 'Buffers hit / read' : ''}</span>
          </div>
          <JsonPlanNode node={plan.root} total={total} analyzed={plan.analyzed} level={0} />
        </div>
      )}
    </div>
  )
}

function JsonPlanNode({
  node,
  total,
  analyzed,
  level,
}: {
  node: JsonNode
  total: number
  analyzed: boolean
  level: number
}) {
  const [expanded, setExpanded] = useState(true)
  const self = analyzed ? (node.selfMs ?? 0) : node.selfCost
  const share = total > 0 ? Math.min(1, self / total) : 0
  const hot = share >= HOT_SHARE
  const misestimated = node.warnings.some((w) => w.startsWith('Rows '))

  return (
    <div>
      <div
        className="grid items-start border-b border-surface-200/50 px-2 py-1 hover:bg-surface-100/60 dark:border-surface-800/50 dark:hover:bg-surface-800/40"
        style={{ gridTemplateColumns: GRID }}
      >
        <div className="flex min-w-0 items-start gap-1" style={{ paddingLeft: level * 16 }}>
          {node.children.length > 0 ? (
            <button
              type="button"
              className="mt-0.5 flex h-4 w-4 flex-shrink-0 items-center justify-center rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              onClick={() => setExpanded((e) => !e)}
              aria-label={expanded ? 'Collapse' : 'Expand'}
            >
              <Icon name={expanded ? 'chevron-down' : 'chevron-right'} className="h-2 w-2" />
            </button>
          ) : (
            <span className="h-4 w-4 flex-shrink-0" />
          )}
          <div className="min-w-0">
            <div className={`font-mono text-xs font-semibold ${hot ? 'text-danger' : 'text-gray-800 dark:text-gray-200'}`}>
              {node.label}
            </div>
            {node.details.map((d) => (
              <div key={d.label} className="break-words font-mono text-[11px] text-gray-500 dark:text-gray-400">
                <span className="text-gray-400 dark:text-gray-500">{d.label}:</span> {d.value}
              </div>
            ))}
            {node.warnings.map((w) => (
              <div key={w} className="mt-0.5 flex items-start gap-1 text-[11px] text-amber-700 dark:text-amber-400">
                <span aria-hidden="true">⚠</span>
                <span>{w}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="pr-3" title={`${(share * 100).toFixed(1)}% of the ${analyzed ? 'total time' : 'total cost'}`}>
          <div className="font-mono text-[11px] text-gray-700 dark:text-gray-300">
            {analyzed ? formatMs(self) : self.toLocaleString(undefined, { maximumFractionDigits: 1 })}
            <span className="text-gray-400 dark:text-gray-500"> · {(share * 100).toFixed(0)}%</span>
          </div>
          <div className="mt-0.5 h-1 w-full rounded-full bg-surface-200 dark:bg-surface-800">
            <div
              className={`h-1 rounded-full ${hot ? 'bg-danger' : 'bg-accent-500'}`}
              style={{ width: `${Math.max(share * 100, share > 0 ? 2 : 0)}%` }}
            />
          </div>
        </div>

        <div className={`text-right font-mono text-[11px] ${misestimated ? 'text-amber-700 dark:text-amber-400' : 'text-gray-600 dark:text-gray-400'}`}>
          {analyzed && node.actualRows !== undefined ? (
            <>
              {node.actualRows.toLocaleString()}
              <span className="text-gray-400 dark:text-gray-500"> / {node.planRows.toLocaleString()}</span>
            </>
          ) : (
            node.planRows.toLocaleString()
          )}
        </div>

        <div className="text-right font-mono text-[11px] text-gray-500 dark:text-gray-400">
          {analyzed && node.loops !== undefined ? node.loops.toLocaleString() : ''}
        </div>

        <div className="text-right font-mono text-[11px] text-gray-500 dark:text-gray-400">
          {node.buffers && (
            <>
              {node.buffers.hit.toLocaleString()}
              <span className={node.buffers.read > 0 ? 'text-amber-700 dark:text-amber-400' : 'text-gray-400 dark:text-gray-500'}>
                {' '}/ {node.buffers.read.toLocaleString()}
              </span>
            </>
          )}
        </div>
      </div>

      {expanded &&
        node.children.map((child, i) => (
          <JsonPlanNode key={i} node={child} total={total} analyzed={analyzed} level={level + 1} />
        ))}
    </div>
  )
}

/** Detect if a QueryResult is an EXPLAIN output (single "QUERY PLAN" column). */
export function isExplainResult(columns: string[]): boolean {
  return columns.length === 1 && columns[0] === 'QUERY PLAN'
}
