import type { Story, StoryDefault } from '@ladle/react'
import { ConnectionDialog } from './ConnectionDialog'

export default {
  title: 'Connection/ConnectionDialog',
} satisfies StoryDefault

export const Default: Story = () => <ConnectionDialog />
