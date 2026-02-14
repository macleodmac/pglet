import { Sidebar } from './Sidebar'
import { StatusBar } from './StatusBar'
import { TabBar } from '../tabs/TabBar'
import { TabPanel } from '../tabs/TabPanel'
import { useTabStore } from '../../stores/tabs'
import { useState, useEffect } from 'react'
import { Group, Panel, Separator, useDefaultLayout } from 'react-resizable-panels'

export function AppShell() {
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const activeTab = tabs.find((t) => t.id === activeTabId)
  const [sidebarSection, setSidebarSection] = useState<'objects' | 'saved' | 'history'>('objects')

  const sidebarLayout = useDefaultLayout({ id: 'pglet-sidebar' })

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'o') {
        e.preventDefault()
        setSidebarSection('saved')
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <Group
        orientation="horizontal"
        className="min-h-0 flex-1"
        defaultLayout={sidebarLayout.defaultLayout}
        onLayoutChanged={sidebarLayout.onLayoutChanged}
      >
        <Panel id="sidebar" defaultSize="18%" minSize="10%" maxSize="35%">
          <Sidebar activeSection={sidebarSection} onSectionChange={setSidebarSection} />
        </Panel>
        <Separator className="w-px bg-surface-200 transition-colors data-[separator=hover]:bg-accent-500 data-[separator=active]:bg-accent-500 dark:bg-surface-800" />
        <Panel id="main" minSize="50%">
          <div className="flex h-full min-w-0 flex-col">
            <TabBar />
            <main className="flex min-h-0 min-w-0 flex-1 flex-col">
              {activeTab && <TabPanel tab={activeTab} />}
            </main>
          </div>
        </Panel>
      </Group>
      <StatusBar />
    </div>
  )
}
