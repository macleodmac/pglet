import type { Meta, StoryObj } from '@storybook/react'

import { AiChatPanel } from './AiChatPanel'

const meta: Meta<typeof AiChatPanel> = {
  title: 'AI/AiChatPanel',
  component: AiChatPanel,
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    (Story) => (
      <div style={{ height: 600, width: 400 }}>
        <Story />
      </div>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof AiChatPanel>

const noop = () => {}

export const Welcome: Story = {
  args: {
    turns: [],
    currentTurnIndex: 0,
    prompt: '',
    error: null,
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const WelcomeWithSuggestions: Story = {
  args: {
    turns: [],
    currentTurnIndex: 0,
    prompt: '',
    error: null,
    isGenerating: false,
    suggestions: [
      'Count orders by status',
      'Find users without orders',
      'Top 10 products by revenue',
      'Average order value per month',
    ],
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const WelcomeWithDraft: Story = {
  args: {
    turns: [],
    currentTurnIndex: 0,
    prompt: 'Show all users who signed up this month',
    error: null,
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const Generating: Story = {
  args: {
    turns: [],
    currentTurnIndex: 0,
    prompt: '',
    error: null,
    isGenerating: true,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const SingleTurn: Story = {
  args: {
    turns: [
      {
        prompt: 'Show all users who signed up this month',
        sql: 'SELECT * FROM users WHERE created_at >= date_trunc(\'month\', now())',
        explanation:
          'This query selects all columns from the users table where the created_at timestamp is on or after the start of the current month.',
      },
    ],
    currentTurnIndex: 0,
    prompt: '',
    error: null,
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const MultipleTurns: Story = {
  args: {
    turns: [
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
    ],
    currentTurnIndex: 2,
    prompt: '',
    error: null,
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const MiddleTurn: Story = {
  args: {
    ...MultipleTurns.args,
    currentTurnIndex: 1,
  },
}

export const WithError: Story = {
  args: {
    turns: [
      {
        prompt: 'Show data from nonexistent_table',
        sql: '',
        explanation: '',
      },
    ],
    currentTurnIndex: 0,
    prompt: '',
    error: 'Failed to generate SQL: table "nonexistent_table" does not exist in the schema.',
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const NoExplanation: Story = {
  args: {
    turns: [
      {
        prompt: 'SELECT 1',
        sql: 'SELECT 1',
        explanation: '',
      },
    ],
    currentTurnIndex: 0,
    prompt: '',
    error: null,
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const LongSql: Story = {
  args: {
    turns: [
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
    ],
    currentTurnIndex: 0,
    prompt: '',
    error: null,
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}

export const ErrorWithSql: Story = {
  args: {
    turns: [
      {
        prompt: 'Show all data from the analytics table',
        sql: 'SELECT * FROM analytics',
        explanation: '',
      },
    ],
    currentTurnIndex: 0,
    prompt: '',
    error: 'relation "analytics" does not exist. Did you mean "user_analytics"?',
    isGenerating: false,
    onPromptChange: noop,
    onSubmit: noop,
    onNavigate: noop,
  },
}
