import { useEffect, useRef, useState } from 'react'
import { useConnectionStore } from './stores/connection'
import { useTabStore } from './stores/tabs'
import { ConnectionDialog } from './components/connection/ConnectionDialog'
import { AppShell } from './components/layout/AppShell'
import { getConnectionInfo } from './api/client'
import icon from './assets/icon.png'

export default function App() {
  const connected = useConnectionStore((s) => s.connected)
  const setConnected = useConnectionStore((s) => s.setConnected)
  const initTabs = useTabStore((s) => s.initFromServer)
  const [checking, setChecking] = useState(true)
  const [connecting, setConnecting] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval>>(null)

  useEffect(() => {
    getConnectionInfo().then((res) => {
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
