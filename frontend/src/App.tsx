import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useConnectionStore } from './stores/connection'
import { useTabStore } from './stores/tabs'
import { ConnectionDialog } from './components/connection/ConnectionDialog'
import { AppShell } from './components/layout/AppShell'
import { getConnectionInfo, useAuthStore } from './api/client'
import { claimWindow, useWindowStore } from './stores/window'
import icon from './assets/icon.png'

function Notice({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex h-screen flex-col items-center justify-center gap-3 bg-white px-6 text-center dark:bg-surface-950">
      <img src={icon} alt="pglet" className="h-14 w-14 opacity-70" />
      <h1 className="text-sm font-semibold text-gray-800 dark:text-gray-200">{title}</h1>
      <div className="max-w-md text-sm text-gray-500 dark:text-gray-400">{children}</div>
    </div>
  )
}

export default function App() {
  const connected = useConnectionStore((s) => s.connected)
  const setConnected = useConnectionStore((s) => s.setConnected)
  const initTabs = useTabStore((s) => s.initFromServer)
  const [checking, setChecking] = useState(true)
  const [connecting, setConnecting] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval>>(null)
  const signedOut = useAuthStore((s) => s.signedOut)
  const inactive = useWindowStore((s) => s.inactive)

  useEffect(() => {
    claimWindow().then(() => getConnectionInfo()).then((res) => {
      const info = res.data
      if (info?.host) {
        setConnected(info)
        return initTabs()
      }
      if (info?.connecting) {
        setConnecting(true)
      }
    }).finally(() => setChecking(false))
  }, [setConnected, initTabs])

  // Poll while server is connecting in the background
  useEffect(() => {
    if (!connecting) return
    intervalRef.current = setInterval(async () => {
      const res = await getConnectionInfo()
      const info = res.data
      if (info?.host) {
        setConnecting(false)
        setConnected(info)
        initTabs()
      } else if (!info?.connecting) {
        // Connection attempt finished but failed
        setConnecting(false)
      }
    }, 500)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [connecting, setConnected, initTabs])

  if (signedOut) {
    return (
      <Notice title="Sign in to pglet">
        Open pglet with the link it printed in its terminal when it started. The link includes a token that
        keeps other websites and users from reaching your database.
      </Notice>
    )
  }

  if (inactive) {
    return (
      <Notice title="pglet is open in another window">
        <p>Only one window can use pglet at a time.</p>
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="mt-3 rounded bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600"
        >
          Use pglet here
        </button>
      </Notice>
    )
  }

  if (checking || connecting) {
    return (
      <div className="flex h-screen flex-col items-center justify-center bg-white dark:bg-surface-950">
        <img src={icon} alt="pglet" className="h-20 w-20 animate-pulse opacity-70" />
        {connecting && <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">Connecting...</p>}
      </div>
    )
  }

  if (!connected) {
    return <ConnectionDialog />
  }

  return <AppShell />
}
