import type { Story, StoryDefault } from '@ladle/react'
import { DataGrid } from './DataGrid'

export default {
  title: 'Results/DataGrid',
  decorators: [
    (Component) => (
      <div style={{ height: '500px' }}>
        <Component />
      </div>
    ),
  ],
} satisfies StoryDefault

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

export const Empty: Story = () => <DataGrid columns={[]} columnTypes={[]} rows={[]} />

export const FewRows: Story = () => <DataGrid columns={sampleColumns} columnTypes={sampleTypes} rows={generateRows(5)} />

export const ManyRows: Story = () => (
  <DataGrid columns={sampleColumns} columnTypes={sampleTypes} rows={generateRows(1000)} />
)

export const WithNulls: Story = () => (
  <DataGrid
    columns={['id', 'name', 'bio', 'avatar_url']}
    columnTypes={['INT4', 'TEXT', 'TEXT', 'TEXT']}
    rows={[
      ['1', 'Alice', 'Software engineer', null],
      ['2', 'Bob', null, 'https://example.com/bob.jpg'],
      ['3', null, null, null],
      ['4', 'Diana', 'Designer', 'https://example.com/diana.jpg'],
    ]}
  />
)

export const WideTable: Story = () => (
  <DataGrid
    columns={Array.from({ length: 20 }, (_, i) => `column_${i + 1}`)}
    columnTypes={Array.from({ length: 20 }, () => 'TEXT')}
    rows={Array.from({ length: 50 }, (_, rowIdx) =>
      Array.from({ length: 20 }, (_, colIdx) => `row${rowIdx + 1}_col${colIdx + 1}`),
    )}
  />
)
