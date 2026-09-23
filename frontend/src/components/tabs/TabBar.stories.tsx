import type { Story, StoryDefault } from '@ladle/react'
import { TabBar } from './TabBar'
import { useTabStore } from '../../stores/tabs'
import { useEffect } from 'react'

export default {
  title: 'Tabs/TabBar',
} satisfies StoryDefault

export const SingleTab: Story = () => <TabBar />

export const MultipleTabs: Story = () => {
  const addQueryTab = useTabStore((s) => s.addQueryTab)
  const addTableTab = useTabStore((s) => s.addTableTab)
  useEffect(() => {
    addQueryTab()
    addTableTab('users')
    addTableTab('orders')
  }, [addQueryTab, addTableTab])
  return <TabBar />
}
