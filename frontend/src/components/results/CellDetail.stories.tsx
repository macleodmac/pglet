import type { Story, StoryDefault } from '@ladle/react'
import { CellDetail } from './CellDetail'

export default {
  title: 'Results/CellDetail',
} satisfies StoryDefault

const noop = () => {}

export const TextValue: Story = () => (
  <CellDetail column="email" columnType="TEXT" value="alice@example.com" onClose={noop} />
)

export const NullValue: Story = () => <CellDetail column="bio" columnType="TEXT" value={null} onClose={noop} />

export const LongText: Story = () => (
  <CellDetail
    column="description"
    columnType="TEXT"
    value={'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(20)}
    onClose={noop}
  />
)

export const JsonValue: Story = () => (
  <CellDetail
    column="metadata"
    columnType="JSONB"
    value={JSON.stringify({
      role: 'admin',
      permissions: ['read', 'write', 'delete'],
      profile: { theme: 'dark', timezone: 'UTC' },
    })}
    onClose={noop}
  />
)

export const JsonArray: Story = () => (
  <CellDetail
    column="tags"
    columnType="JSONB"
    value={JSON.stringify(['postgres', 'database', 'sql', 'admin'])}
    onClose={noop}
  />
)

export const EmptyString: Story = () => <CellDetail column="notes" columnType="TEXT" value="" onClose={noop} />

export const Uuid: Story = () => (
  <CellDetail column="id" columnType="UUID" value="a1b2c3d4-e5f6-7890-abcd-ef1234567890" onClose={noop} />
)

export const Timestamp: Story = () => (
  <CellDetail column="created_at" columnType="TIMESTAMPTZ" value="2025-01-15T09:30:00.000Z" onClose={noop} />
)

export const Integer: Story = () => <CellDetail column="order_count" columnType="INT4" value="42" onClose={noop} />
