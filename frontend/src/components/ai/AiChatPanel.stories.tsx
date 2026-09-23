import type { Story, StoryDefault } from '@ladle/react'
import type { ComponentProps } from 'react'
import { AiChatPanel } from './AiChatPanel'

export default {
  title: 'AI/AiChatPanel',
  decorators: [
    (Component) => (
      <div style={{ height: 600, width: 400 }}>
        <Component />
      </div>
    ),
  ],
} satisfies StoryDefault

type Props = ComponentProps<typeof AiChatPanel>

const noop = () => {}

const base: Props = {
  turns: [],
  currentTurnIndex: 0,
  prompt: '',
  error: null,
  isGenerating: false,
  onPromptChange: noop,
  onSubmit: noop,
  onNavigate: noop,
}

const multipleTurns: Props['turns'] = [
  {
    prompt: 'Show all users who signed up this month',
    sql: 'SELECT * FROM users WHERE created_at >= date_trunc(\'month\', now())',
    explanation: 'Selects all users created since the start of the current month.',
  },
  {
    prompt: 'Only show name and email, order by newest first',
    sql: 'SELECT name, email FROM users WHERE created_at >= date_trunc(\'month\', now()) ORDER BY created_at DESC',
    explanation:
      'Refined to only return name and email columns, ordered by creation date descending.',
  },
  {
    prompt: 'Add a count of their orders',
    sql: 'SELECT u.name, u.email, COUNT(o.id) AS order_count FROM users u LEFT JOIN orders o ON o.user_id = u.id WHERE u.created_at >= date_trunc(\'month\', now()) GROUP BY u.id, u.name, u.email ORDER BY u.created_at DESC',
    explanation:
      'Added a LEFT JOIN to the orders table and grouped by user to count orders per user.',
  },
]

export const Welcome: Story = () => <AiChatPanel {...base} />

export const WelcomeWithSuggestions: Story = () => (
  <AiChatPanel
    {...base}
    suggestions={[
      'Count orders by status',
      'Find users without orders',
      'Top 10 products by revenue',
      'Average order value per month',
    ]}
  />
)

export const WelcomeWithDraft: Story = () => (
  <AiChatPanel {...base} prompt="Show all users who signed up this month" />
)

export const Generating: Story = () => <AiChatPanel {...base} isGenerating />

export const SingleTurn: Story = () => (
  <AiChatPanel
    {...base}
    turns={[
      {
        prompt: 'Show all users who signed up this month',
        sql: 'SELECT * FROM users WHERE created_at >= date_trunc(\'month\', now())',
        explanation:
          'This query selects all columns from the users table where the created_at timestamp is on or after the start of the current month.',
      },
    ]}
  />
)

export const MultipleTurns: Story = () => <AiChatPanel {...base} turns={multipleTurns} currentTurnIndex={2} />

export const MiddleTurn: Story = () => <AiChatPanel {...base} turns={multipleTurns} currentTurnIndex={1} />

export const WithError: Story = () => (
  <AiChatPanel
    {...base}
    turns={[{ prompt: 'Show data from nonexistent_table', sql: '', explanation: '' }]}
    error={'Failed to generate SQL: table "nonexistent_table" does not exist in the schema.'}
  />
)

export const NoExplanation: Story = () => (
  <AiChatPanel {...base} turns={[{ prompt: 'SELECT 1', sql: 'SELECT 1', explanation: '' }]} />
)

export const LongSql: Story = () => (
  <AiChatPanel
    {...base}
    turns={[
      {
        prompt: 'Show comprehensive user analytics with order history, lifetime value, and activity metrics',
        sql: `SELECT
  u.id,
  u.name,
  u.email,
  u.created_at AS signup_date,
  COUNT(DISTINCT o.id) AS total_orders,
  COALESCE(SUM(o.total), 0) AS lifetime_value,
  COALESCE(AVG(o.total), 0) AS avg_order_value,
  MAX(o.created_at) AS last_order_date,
  COUNT(DISTINCT DATE(o.created_at)) AS active_days,
  EXTRACT(DAY FROM NOW() - MAX(o.created_at)) AS days_since_last_order
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.created_at >= date_trunc('year', now())
GROUP BY u.id, u.name, u.email, u.created_at
HAVING COUNT(o.id) > 0
ORDER BY lifetime_value DESC
LIMIT 50`,
        explanation:
          'Comprehensive analytics query joining users with their orders, calculating lifetime value, average order, and recency metrics. Filtered to users who signed up this year and have at least one order.',
      },
    ]}
  />
)

export const ErrorWithSql: Story = () => (
  <AiChatPanel
    {...base}
    turns={[{ prompt: 'Show all data from the analytics table', sql: 'SELECT * FROM analytics', explanation: '' }]}
    error={'relation "analytics" does not exist. Did you mean "user_analytics"?'}
  />
)
