import type { Meta, StoryObj } from '@storybook/react'
import { TabBar } from './TabBar'
import { useTabStore } from '../../stores/tabs'
import { useEffect } from 'react'

const meta: Meta<typeof TabBar> = {
  title: 'Tabs/TabBar',
  component: TabBar,
  parameters: {
    layout: 'fullscreen',
  },
}

export default meta
type Story = StoryObj<typeof TabBar>

export const SingleTab: Story = {}

export const MultipleTabs: Story = {
  decorators: [
    (Story) => {
      const addQueryTab = useTabStore((s) => s.addQueryTab)
      const addTableTab = useTabStore((s) => s.addTableTab)
      useEffect(() => {
        addQueryTab()
        addTableTab('users')
        addTableTab('orders')
      }, [])
      return <Story />
    },
  ],
}
