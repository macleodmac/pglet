/**
 * Turns EXPLAIN (FORMAT JSON) output into a tree with per-node self time or
 * self cost and warnings about the usual suspects: hot nodes, bad row
 * estimates, big sequential scans and operations that spilled to disk.
 */

/** One node as Postgres emits it; only the fields read here are typed. */
interface RawNode {
  'Node Type': string
  'Parent Relationship'?: string
  'Subplan Name'?: string
  'Relation Name'?: string
  Schema?: string
  Alias?: string
  'Index Name'?: string
  'CTE Name'?: string
  'Function Name'?: string
  'Join Type'?: string
  Strategy?: string
  'Startup Cost': number
  'Total Cost': number
  'Plan Rows': number
  'Actual Total Time'?: number
  'Actual Rows'?: number
  'Actual Loops'?: number
  'Rows Removed by Filter'?: number
  'Rows Removed by Index Recheck'?: number
  'Workers Launched'?: number
  'Sort Space Type'?: string
  'Sort Space Used'?: number
  'Hash Batches'?: number
  'Original Hash Batches'?: number
  'Shared Hit Blocks'?: number
  'Shared Read Blocks'?: number
  'Temp Written Blocks'?: number
  Plans?: RawNode[]
  [key: string]: unknown
}

interface RawPlan {
  Plan: RawNode
  'Planning Time'?: number
  'Execution Time'?: number
}

export interface PlanDetail {
  label: string
  value: string
}

export interface PlanNode {
  /** e.g. "Index Scan using people_pkey on people p" */
  label: string
  nodeType: string
  details: PlanDetail[]
  warnings: string[]
  children: PlanNode[]

  totalCost: number
  /** Cost of this node excluding its children. */
  selfCost: number
  /** Estimated rows per loop. */
  planRows: number

  // Present only for EXPLAIN ANALYZE.
  /** Rows per loop, as Postgres reports them. */
  actualRows?: number
  loops?: number
  /** Wall time including children, in ms. */
  totalMs?: number
  /** Wall time excluding children, in ms. */
  selfMs?: number
  /** Shared buffers hit and read, including children. */
  buffers?: { hit: number; read: number }
}

export interface ParsedPlan {
  root: PlanNode
  analyzed: boolean
  planningMs?: number
  executionMs?: number
  totalCost: number
}

// Thresholds for warnings.
const MISESTIMATE_FACTOR = 10
const MISESTIMATE_MIN_ROWS = 100
const BIG_SEQ_SCAN_ROWS = 10_000
/** Seq scans keeping less than this share of rows read are flagged. */
const SELECTIVE_FILTER_SHARE = 0.1
/** A node is "hot" when it accounts for at least this share of the total. */
export const HOT_SHARE = 0.3

/** Detail fields worth showing, in display order. */
const DETAIL_KEYS = [
  'Index Cond',
  'Recheck Cond',
  'Hash Cond',
  'Merge Cond',
  'Join Filter',
  'Filter',
  'Rows Removed by Filter',
  'Rows Removed by Join Filter',
  'Rows Removed by Index Recheck',
  'Sort Key',
  'Sort Method',
  'Group Key',
  'Cache Key',
  'Heap Fetches',
  'Hash Buckets',
  'Hash Batches',
  'Peak Memory Usage',
  'Workers Planned',
  'Workers Launched',
]

function formatValue(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  if (typeof v === 'number') return v.toLocaleString()
  return String(v)
}

function nodeLabel(n: RawNode): string {
  let label = n['Node Type']
  if (n.Strategy && n['Node Type'] === 'Aggregate' && n.Strategy !== 'Plain') label = `${n.Strategy} ${label}`
  if (n['Join Type'] && n['Join Type'] !== 'Inner') label += ` (${n['Join Type']})`
  if (n['Index Name']) label += ` using ${n['Index Name']}`

  const target = n['Relation Name'] ?? n['CTE Name'] ?? n['Function Name']
  if (target) {
    const qualified = n.Schema && n.Schema !== 'public' ? `${n.Schema}.${target}` : target
    label += ` on ${qualified}`
    if (n.Alias && n.Alias !== target) label += ` ${n.Alias}`
  } else if (n['Node Type'] === 'Subquery Scan' && n.Alias) {
    label += ` on ${n.Alias}`
  }

  const rel = n['Parent Relationship']
  if (rel === 'InitPlan' || rel === 'SubPlan') label = `${n['Subplan Name'] ?? rel}: ${label}`
  return label
}

function nodeDetails(n: RawNode): PlanDetail[] {
  const details: PlanDetail[] = []
  for (const key of DETAIL_KEYS) {
    const v = n[key]
    if (v === undefined || v === null || v === '') continue
    if (key === 'Sort Method' && n['Sort Space Used'] !== undefined) {
      details.push({ label: key, value: `${v} (${n['Sort Space Type']}: ${n['Sort Space Used']} kB)` })
    } else if (typeof v === 'number' && v === 0 && key.startsWith('Rows Removed')) {
      continue
    } else {
      details.push({ label: key, value: formatValue(v) })
    }
  }
  return details
}

interface BuildContext {
  analyzed: boolean
  /**
   * Processes sharing this node's loops (workers + leader under a Gather).
   * Postgres reports time averaged per loop, so wall time is
   * time * loops / processes.
   */
  parallelism: number
  /** Below a Limit, nodes stop early, so fewer rows than estimated is expected. */
  underLimit: boolean
}

function build(n: RawNode, ctx: BuildContext): PlanNode {
  const workers = n['Workers Launched']
  const isGather = n['Node Type'] === 'Gather' || n['Node Type'] === 'Gather Merge'
  const childCtx: BuildContext = {
    analyzed: ctx.analyzed,
    parallelism: isGather && workers !== undefined ? workers + 1 : ctx.parallelism,
    underLimit: ctx.underLimit || n['Node Type'] === 'Limit',
  }
  const children = (n.Plans ?? []).map((c) => build(c, childCtx))

  const childCost = children.reduce((sum, c) => sum + c.totalCost, 0)
  const node: PlanNode = {
    label: nodeLabel(n),
    nodeType: n['Node Type'],
    details: nodeDetails(n),
    warnings: [],
    children,
    totalCost: n['Total Cost'],
    selfCost: Math.max(0, n['Total Cost'] - childCost),
    planRows: n['Plan Rows'],
  }

  if (ctx.analyzed && n['Actual Total Time'] !== undefined) {
    const loops = n['Actual Loops'] ?? 1
    node.loops = loops
    node.actualRows = n['Actual Rows'] ?? 0
    // A node that never ran (loops = 0) took no time.
    node.totalMs = (n['Actual Total Time'] * loops) / Math.max(1, Math.min(ctx.parallelism, loops))
    const childMs = children.reduce((sum, c) => sum + (c.totalMs ?? 0), 0)
    node.selfMs = Math.max(0, node.totalMs - childMs)
    if (n['Shared Hit Blocks'] !== undefined) {
      node.buffers = { hit: n['Shared Hit Blocks'] ?? 0, read: n['Shared Read Blocks'] ?? 0 }
    }
  }

  node.warnings = nodeWarnings(n, node, ctx.underLimit)
  return node
}

function percent(share: number): string {
  return share < 0.001 ? '<0.1%' : `${(share * 100).toFixed(1)}%`
}

function nodeWarnings(n: RawNode, node: PlanNode, underLimit: boolean): string[] {
  const warnings: string[] = []
  const isSeqScan = n['Node Type'] === 'Seq Scan' || n['Node Type'] === 'Parallel Seq Scan'

  if (node.actualRows !== undefined && node.loops) {
    const est = Math.max(node.planRows, 1)
    const actual = Math.max(node.actualRows, 1)
    const factor = Math.max(est / actual, actual / est)
    const over = est > actual
    if (
      factor >= MISESTIMATE_FACTOR &&
      Math.max(node.planRows, node.actualRows) >= MISESTIMATE_MIN_ROWS &&
      !(over && underLimit)
    ) {
      const dir = over ? 'over' : 'under'
      warnings.push(
        `Rows ${dir}estimated ${Math.round(factor).toLocaleString()}× (estimated ${node.planRows.toLocaleString()}, actual ${node.actualRows.toLocaleString()})`,
      )
    }
    // A full scan is only worth flagging when the filter throws most of it
    // away; reading every row of a table you need every row of is fine.
    const removed = n['Rows Removed by Filter'] ?? 0
    if (isSeqScan && removed > 0) {
      const read = (node.actualRows + removed) * node.loops
      const kept = (node.actualRows * node.loops) / read
      if (read >= BIG_SEQ_SCAN_ROWS && kept < SELECTIVE_FILTER_SHARE) {
        warnings.push(
          `Sequential scan read ${read.toLocaleString()} rows to keep ${percent(kept)}; an index on the filtered columns may help`,
        )
      }
    }
  }

  if (n['Sort Space Type'] === 'Disk') {
    warnings.push(`Sort spilled to disk (${(n['Sort Space Used'] ?? 0).toLocaleString()} kB); consider raising work_mem`)
  }
  if ((n['Hash Batches'] ?? 1) > 1) {
    warnings.push(`Hash spilled to disk in ${n['Hash Batches']} batches; consider raising work_mem`)
  }
  return warnings
}

function isRawPlanArray(v: unknown): v is RawPlan[] {
  return (
    Array.isArray(v) &&
    v.length > 0 &&
    typeof v[0] === 'object' &&
    v[0] !== null &&
    'Plan' in v[0] &&
    typeof (v[0] as RawPlan).Plan?.['Node Type'] === 'string'
  )
}

/** Parses EXPLAIN (FORMAT JSON) text; returns null for anything else. */
export function parsePlanJson(text: string): ParsedPlan | null {
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return null
  }
  if (!isRawPlanArray(parsed)) return null

  const raw = parsed[0]
  const analyzed = raw.Plan['Actual Total Time'] !== undefined
  return {
    root: build(raw.Plan, { analyzed, parallelism: 1, underLimit: false }),
    analyzed,
    planningMs: raw['Planning Time'],
    executionMs: raw['Execution Time'],
    totalCost: raw.Plan['Total Cost'],
  }
}

/** Counts nodes and warnings across the tree. */
export function planStats(node: PlanNode): { nodes: number; warnings: number } {
  return node.children.reduce(
    (acc, c) => {
      const s = planStats(c)
      return { nodes: acc.nodes + s.nodes, warnings: acc.warnings + s.warnings }
    },
    { nodes: 1, warnings: node.warnings.length },
  )
}
