import { useEffect, useState } from 'react'
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

  useEffect(() => {
    getConnectionInfo().then((res) => {
      const info = res.data
      if (info?.host) {
        setConnected(info)
        return initTabs()
      }
    }).finally(() => setChecking(false))
  }, [setConnected, initTabs])

  if (checking) {
    return (
      <div className="flex h-screen items-center justify-center bg-white dark:bg-surface-950">
        <img src={icon} alt="pglet" className="h-20 w-20 animate-pulse opacity-70" />
      </div>
    )
  }

  if (!connected) {
    return <ConnectionDialog />
  }

  return <AppShell />
}
