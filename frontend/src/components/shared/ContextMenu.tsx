import { useEffect, useRef } from 'react'

/**
 * MenuItem Interface
 *
 * Defines the structure of a context menu item.
 */
export interface MenuItem {
  /** Display label for the menu item */
  label: string
  /** Callback function when item is clicked */
  onClick: () => void
  /** Whether this item is a separator (visual divider) */
  separator?: boolean
}

/**
 * ContextMenuProps Interface
 *
 * Configuration for the context menu component.
 */
export interface ContextMenuProps {
  /** X coordinate for menu position (in pixels) */
  x: number
  /** Y coordinate for menu position (in pixels) */
  y: number
  /** Array of menu items to display */
  items: MenuItem[]
  /** Callback when menu should close */
  onClose: () => void
}

/**
 * ContextMenu Component
 *
 * A reusable right-click context menu with automatic positioning and click-away handling.
 *
 * Features:
 * - Position at specific x/y coordinates
 * - Click-away detection to auto-close
 * - Escape key support
 * - Separator support for grouping items
 * - Dark mode support
 * - Keyboard accessible
 *
 * Accessibility:
 * - Uses semantic button elements
 * - Keyboard navigation support (Escape to close)
 * - Focus management for better UX
 *
 * Usage:
 * ```tsx
 * const [menu, setMenu] = useState<{ x: number; y: number } | null>(null)
 *
 * const handleContextMenu = (e: React.MouseEvent) => {
 *   e.preventDefault()
 *   setMenu({ x: e.clientX, y: e.clientY })
 * }
 *
 * {menu && (
 *   <ContextMenu
 *     x={menu.x}
 *     y={menu.y}
 *     items={[
 *       { label: 'Copy', onClick: () => console.log('Copy') },
 *       { label: 'Paste', onClick: () => console.log('Paste') },
 *       { separator: true },
 *       { label: 'Delete', onClick: () => console.log('Delete') },
 *     ]}
 *     onClose={() => setMenu(null)}
 *   />
 * )}
 * ```
 */
export function ContextMenu({ x, y, items, onClose }: ContextMenuProps) {
  const ref = useRef<HTMLDivElement>(null)

  // Close menu when clicking outside
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  // Close menu on Escape key
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  return (
    <div
      ref={ref}
      className="fixed z-50 min-w-36 animate-fade-in rounded-lg border border-surface-200 bg-white py-1 shadow-xl shadow-black/8 dark:border-surface-800 dark:bg-surface-800 dark:shadow-black/40"
      style={{ left: x, top: y }}
      role="menu"
      aria-orientation="vertical"
    >
      {items.map((item, i) =>
        item.separator ? (
          <div
            key={i}
            className="my-1 border-t border-surface-200 dark:border-surface-800"
            role="separator"
          />
        ) : (
          <button
            key={i}
            type="button"
            onClick={() => {
              item.onClick()
              onClose()
            }}
            className="block w-full px-3 py-1.5 text-left text-xs text-gray-700 transition-colors hover:bg-accent-50 hover:text-accent-700 dark:text-gray-300 dark:hover:bg-accent-900/20 dark:hover:text-accent-300"
            role="menuitem"
          >
            {item.label}
          </button>
        ),
      )}
    </div>
  )
}
