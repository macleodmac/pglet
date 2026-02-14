import type { Meta, StoryObj } from '@storybook/react'
import { SqlEditor } from './SqlEditor'
import { useState } from 'react'

const meta: Meta<typeof SqlEditor> = {
  title: 'Editor/SqlEditor',
  component: SqlEditor,
  args: {
    onRun: () => console.log('run'),
    onExplain: () => console.log('explain'),
  },
  decorators: [
    (Story) => (
      <div style={{ height: '400px' }}>
        <Story />
      </div>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof SqlEditor>

export const Empty: Story = {
  render: (args) => {
    const [value, setValue] = useState('')
    return <SqlEditor {...args} value={value} onChange={setValue} />
  },
}

export const WithQuery: Story = {
  render: (args) => {
    const [value, setValue] = useState(
      `SELECT u.id, u.name, u.email, count(o.id) as order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.created_at > now() - interval '30 days'
GROUP BY u.id, u.name, u.email
ORDER BY order_count DESC
LIMIT 50;`,
    )
    return <SqlEditor {...args} value={value} onChange={setValue} />
  },
}

export const WithLongQuery: Story = {
  render: (args) => {
    const [value, setValue] = useState(
      `-- Monthly active users report
WITH monthly_events AS (
    SELECT
        user_id,
        date_trunc('month', created_at) as month,
        count(*) as event_count
    FROM events
    WHERE created_at >= now() - interval '12 months'
    GROUP BY user_id, date_trunc('month', created_at)
),
user_stats AS (
    SELECT
        user_id,
        count(DISTINCT month) as active_months,
        sum(event_count) as total_events,
        min(month) as first_active,
        max(month) as last_active
    FROM monthly_events
    GROUP BY user_id
)
SELECT
    u.name,
    u.email,
    us.active_months,
    us.total_events,
    us.first_active,
    us.last_active
FROM user_stats us
JOIN users u ON u.id = us.user_id
WHERE us.active_months >= 3
ORDER BY us.total_events DESC;`,
    )
    return <SqlEditor {...args} value={value} onChange={setValue} />
  },
}
