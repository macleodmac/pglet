import { useState } from 'react'

interface JsonbViewerProps {
  data: unknown
  initialExpanded?: number
}

export function JsonbViewer({ data, initialExpanded = 2 }: JsonbViewerProps) {
  return (
    <div className="font-mono text-xs">
      <JsonNode value={data} depth={0} initialExpanded={initialExpanded} />
    </div>
  )
}

function JsonNode({
  value,
  depth,
  initialExpanded,
  keyName,
}: {
  value: unknown
  depth: number
  initialExpanded: number
  keyName?: string
}) {
  const [expanded, setExpanded] = useState(depth < initialExpanded)

  const indent = depth * 16

  if (value === null) {
    return (
      <div style={{ paddingLeft: indent }}>
        {keyName !== undefined && <KeyLabel name={keyName} />}
        <span className="italic text-gray-400">null</span>
      </div>
    )
  }

  if (typeof value === 'boolean') {
    return (
      <div style={{ paddingLeft: indent }}>
        {keyName !== undefined && <KeyLabel name={keyName} />}
        <span className="text-purple-600 dark:text-purple-400">{String(value)}</span>
      </div>
    )
  }

  if (typeof value === 'number') {
    return (
      <div style={{ paddingLeft: indent }}>
        {keyName !== undefined && <KeyLabel name={keyName} />}
        <span className="text-blue-600 dark:text-blue-400">{value}</span>
      </div>
    )
  }

  if (typeof value === 'string') {
    return (
      <div style={{ paddingLeft: indent }}>
        {keyName !== undefined && <KeyLabel name={keyName} />}
        <span className="text-green-600 dark:text-green-400">"{value}"</span>
      </div>
    )
  }

  if (Array.isArray(value)) {
    if (value.length === 0) {
      return (
        <div style={{ paddingLeft: indent }}>
          {keyName !== undefined && <KeyLabel name={keyName} />}
          <span className="text-gray-500">[]</span>
        </div>
      )
    }
    return (
      <div>
        <div
          style={{ paddingLeft: indent }}
          className="cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800"
          onClick={() => setExpanded(!expanded)}
        >
          {keyName !== undefined && <KeyLabel name={keyName} />}
          <span className="text-gray-500">
            {expanded ? '[' : `[${value.length} items]`}
          </span>
        </div>
        {expanded && (
          <>
            {value.map((item, i) => (
              <JsonNode
                key={i}
                value={item}
                depth={depth + 1}
                initialExpanded={initialExpanded}
                keyName={String(i)}
              />
            ))}
            <div style={{ paddingLeft: indent }} className="text-gray-500">
              ]
            </div>
          </>
        )}
      </div>
    )
  }

  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>)
    if (entries.length === 0) {
      return (
        <div style={{ paddingLeft: indent }}>
          {keyName !== undefined && <KeyLabel name={keyName} />}
          <span className="text-gray-500">{'{}'}</span>
        </div>
      )
    }
    return (
      <div>
        <div
          style={{ paddingLeft: indent }}
          className="cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800"
          onClick={() => setExpanded(!expanded)}
        >
          {keyName !== undefined && <KeyLabel name={keyName} />}
          <span className="text-gray-500">
            {expanded ? '{' : `{${entries.length} keys}`}
          </span>
        </div>
        {expanded && (
          <>
            {entries.map(([k, v]) => (
              <JsonNode
                key={k}
                value={v}
                depth={depth + 1}
                initialExpanded={initialExpanded}
                keyName={k}
              />
            ))}
            <div style={{ paddingLeft: indent }} className="text-gray-500">
              {'}'}
            </div>
          </>
        )}
      </div>
    )
  }

  return (
    <div style={{ paddingLeft: indent }}>
      {keyName !== undefined && <KeyLabel name={keyName} />}
      <span className="text-gray-500">{String(value)}</span>
    </div>
  )
}

function KeyLabel({ name }: { name: string }) {
  return (
    <span className="mr-1.5 text-gray-700 dark:text-gray-300">
      {name}:
    </span>
  )
}
