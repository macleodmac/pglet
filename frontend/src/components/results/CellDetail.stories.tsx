import type { Meta, StoryObj } from '@storybook/react'
import { CellDetail } from './CellDetail'

const meta: Meta<typeof CellDetail> = {
  title: 'Results/CellDetail',
  component: CellDetail,
  parameters: {
    layout: 'fullscreen',
  },
}

export default meta
type Story = StoryObj<typeof CellDetail>

export const TextValue: Story = {
  args: {
    column: 'email',
    columnType: 'TEXT',
    value: 'alice@example.com',
    onClose: () => {},
  },
}

export const NullValue: Story = {
  args: {
    column: 'bio',
    columnType: 'TEXT',
    value: null,
    onClose: () => {},
  },
}

export const LongText: Story = {
  args: {
    column: 'description',
    columnType: 'TEXT',
    value: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(20),
    onClose: () => {},
  },
}

export const JsonValue: Story = {
  args: {
    column: 'metadata',
    columnType: 'JSONB',
    value: JSON.stringify({
      role: 'admin',
      permissions: ['read', 'write', 'delete'],
      profile: { theme: 'dark', timezone: 'UTC' },
    }),
    onClose: () => {},
  },
}

export const JsonArray: Story = {
  args: {
    column: 'tags',
    columnType: 'JSONB',
    value: JSON.stringify(['postgres', 'database', 'sql', 'admin']),
    onClose: () => {},
  },
}
