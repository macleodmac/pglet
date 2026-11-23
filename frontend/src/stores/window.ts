import { create } from 'zustand'
import { basePath } from '../api/client'

/**
 * Only one window may drive pglet at a time: tabs and their transactions are
 * keyed by tab ID on the server, so two windows showing the same tabs would
 * overwrite each other's saved state and act on each other's transactions.
 * The most recently opened (or reclaimed) window wins.
 */
interface WindowState {
  /** Another window took over; this one must not save or run anything */
  inactive: boolean
}

export const useWindowStore = create<WindowState>(() => ({ inactive: false }))

type Message = { type: 'claim'; id: string } | { type: 'released'; to: string }

const windowId = crypto.randomUUID()
const channel = typeof BroadcastChannel === 'undefined' ? null : new BroadcastChannel(`pglet${basePath}`)
let beforeRelease: () => void = () => {}

/** Register work to finish (saving tabs) before another window takes over. */
export function onRelease(fn: () => void) {
  beforeRelease = fn
}

channel?.addEventListener('message', (e: MessageEvent<Message>) => {
  const msg = e.data
  if (msg.type === 'claim' && msg.id !== windowId && !useWindowStore.getState().inactive) {
    beforeRelease()
    useWindowStore.setState({ inactive: true })
    channel.postMessage({ type: 'released', to: msg.id } satisfies Message)
  }
})

/** Become the active window, waiting briefly for the previous one to save and step aside. */
export function claimWindow(): Promise<void> {
  useWindowStore.setState({ inactive: false })
  if (!channel) return Promise.resolve()
  return new Promise((resolve) => {
    const timer = setTimeout(finish, 500)
    function finish() {
      clearTimeout(timer)
      channel!.removeEventListener('message', onMessage)
      resolve()
    }
    function onMessage(e: MessageEvent<Message>) {
      if (e.data.type === 'released' && e.data.to === windowId) {
        // The previous window's save is a keepalive request; give it a moment.
        setTimeout(finish, 100)
      }
    }
    channel.addEventListener('message', onMessage)
    channel.postMessage({ type: 'claim', id: windowId } satisfies Message)
  })
}
