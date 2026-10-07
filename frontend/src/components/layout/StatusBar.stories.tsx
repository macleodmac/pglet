import type { Story, StoryDefault } from '@ladle/react'
import { StatusBar } from './StatusBar'
import { useConnectionStore } from '../../stores/connection'
import { useEffect } from 'react'

export default {
  title: 'Layout/StatusBar',
} satisfies StoryDefault

export const Disconnected: Story = () => {
  const setDisconnected = useConnectionStore((s) => s.setDisconnected)
  useEffect(() => {
    setDisconnected()
  }, [setDisconnected])
  return <StatusBar />
}

export const Connected: Story = () => {
  const setConnected = useConnectionStore((s) => s.setConnected)
  useEffect(() => {
    setConnected({
      host: 'localhost',
      port: 5432,
      user: 'postgres',
      database: 'myapp_development',
      version: '16.2',
    })
  }, [setConnected])
  return <StatusBar />
}

export const ReadOnlyWithTimeout: Story = () => {
  const setConnected = useConnectionStore((s) => s.setConnected)
  useEffect(() => {
    setConnected({
      host: 'prod-replica',
      port: 5432,
      user: 'analyst',
      database: 'myapp_production',
      version: '16.2',
      read_only: true,
      statement_timeout_ms: 30_000,
    })
  }, [setConnected])
  return <StatusBar />
}
