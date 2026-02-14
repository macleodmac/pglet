import type { Meta, StoryObj } from '@storybook/react'
import { StatusBar } from './StatusBar'
import { useConnectionStore } from '../../stores/connection'
import { useEffect } from 'react'

const meta: Meta<typeof StatusBar> = {
  title: 'Layout/StatusBar',
  component: StatusBar,
  parameters: {
    layout: 'fullscreen',
  },
}

export default meta
type Story = StoryObj<typeof StatusBar>

export const Disconnected: Story = {
  decorators: [
    (Story) => {
      const setDisconnected = useConnectionStore((s) => s.setDisconnected)
      useEffect(() => { setDisconnected() }, [])
      return <Story />
    },
  ],
}

export const Connected: Story = {
  decorators: [
    (Story) => {
      const setConnected = useConnectionStore((s) => s.setConnected)
      useEffect(() => {
        setConnected({
          host: 'localhost',
          port: 5432,
          user: 'postgres',
          database: 'myapp_development',
          version: '16.2',
        })
      }, [])
      return <Story />
    },
  ],
}
