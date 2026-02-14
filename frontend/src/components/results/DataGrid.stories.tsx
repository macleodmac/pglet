import type { Meta, StoryObj } from '@storybook/react'
import { DataGrid } from './DataGrid'

const meta: Meta<typeof DataGrid> = {
  title: 'Results/DataGrid',
  component: DataGrid,
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    (Story) => (
      <div style={{ height: '500px' }}>
        <Story />
      </div>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof DataGrid>

const sampleColumns = ['id', 'name', 'email', 'created_at', 'metadata']
const sampleTypes = ['INT4', 'TEXT', 'TEXT', 'TIMESTAMPTZ', 'JSONB']

function generateRows(count: number): (string | null)[][] {
  return Array.from({ length: count }, (_, i) => [
    String(i + 1),
    `User ${i + 1}`,
    `user${i + 1}@example.com`,
    new Date(2024, 0, 1 + i).toISOString(),
    i % 5 === 0 ? null : JSON.stringify({ role: i % 2 === 0 ? 'admin' : 'user', loginCount: i * 3 }),
  ])
}

export const Empty: Story = {
  args: {
    columns: [],
    columnTypes: [],
    rows: [],
  },
}

export const FewRows: Story = {
  args: {
    columns: sampleColumns,
    columnTypes: sampleTypes,
    rows: generateRows(5),
  },
}

export const ManyRows: Story = {
  args: {
    columns: sampleColumns,
    columnTypes: sampleTypes,
    rows: generateRows(1000),
  },
}

export const WithNulls: Story = {
  args: {
    columns: ['id', 'name', 'bio', 'avatar_url'],
    columnTypes: ['INT4', 'TEXT', 'TEXT', 'TEXT'],
    rows: [
      ['1', 'Alice', 'Software engineer', null],
      ['2', 'Bob', null, 'https://example.com/bob.jpg'],
      ['3', null, null, null],
      ['4', 'Diana', 'Designer', 'https://example.com/diana.jpg'],
    ],
  },
}

export const WideTable: Story = {
  args: {
    columns: Array.from({ length: 20 }, (_, i) => `column_${i + 1}`),
    columnTypes: Array.from({ length: 20 }, () => 'TEXT'),
    rows: Array.from({ length: 50 }, (_, rowIdx) =>
      Array.from({ length: 20 }, (_, colIdx) => `row${rowIdx + 1}_col${colIdx + 1}`),
    ),
  },
}
