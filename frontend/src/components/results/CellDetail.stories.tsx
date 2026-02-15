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

export const EmptyString: Story = {
  args: {
    column: 'notes',
    columnType: 'TEXT',
    value: '',
    onClose: () => {},
  },
}

export const Uuid: Story = {
  args: {
    column: 'id',
    columnType: 'UUID',
    value: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    onClose: () => {},
  },
}

export const Timestamp: Story = {
  args: {
    column: 'created_at',
    columnType: 'TIMESTAMPTZ',
    value: '2025-01-15T09:30:00.000Z',
    onClose: () => {},
  },
}

export const Integer: Story = {
  args: {
    column: 'order_count',
    columnType: 'INT4',
    value: '42',
    onClose: () => {},
  },
}
