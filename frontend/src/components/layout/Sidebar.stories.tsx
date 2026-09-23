import type { Story, StoryDefault } from '@ladle/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { Sidebar } from './Sidebar'
import type { SchemaGroup } from '../../api/generated'

const mockObjects: Record<string, SchemaGroup> = {
  public: {
    tables: [
      { name: 'users', schema: 'public', type: 'table' },
      { name: 'orders', schema: 'public', type: 'table' },
      { name: 'products', schema: 'public', type: 'table' },
      { name: 'categories', schema: 'public', type: 'table' },
      { name: 'order_items', schema: 'public', type: 'table' },
    ],
    views: [
      { name: 'active_users', schema: 'public', type: 'view' },
      { name: 'order_summary', schema: 'public', type: 'view' },
    ],
    materialized_views: [
      { name: 'monthly_stats', schema: 'public', type: 'materialized_view' },
    ],
    functions: [
      { name: 'calculate_total', schema: 'public', type: 'function' },
      { name: 'update_timestamp', schema: 'public', type: 'function' },
    ],
    sequences: [
      { name: 'users_id_seq', schema: 'public', type: 'sequence' },
      { name: 'orders_id_seq', schema: 'public', type: 'sequence' },
    ],
    types: [],
  },
  auth: {
    tables: [
      { name: 'sessions', schema: 'auth', type: 'table' },
      { name: 'tokens', schema: 'auth', type: 'table' },
    ],
    views: [],
    materialized_views: [],
    functions: [
      { name: 'verify_token', schema: 'auth', type: 'function' },
    ],
    sequences: [],
    types: [],
  },
}

function SidebarWithClient({ createClient }: { createClient: () => QueryClient }) {
  const [client] = useState(createClient)
  return (
    <QueryClientProvider client={client}>
      <div style={{ height: '600px', display: 'flex' }}>
        <Sidebar />
      </div>
    </QueryClientProvider>
  )
}

export default {
  title: 'Layout/Sidebar',
} satisfies StoryDefault

export const WithData: Story = () => (
  <SidebarWithClient
    createClient={() => {
      const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
      qc.setQueryData(['objects'], mockObjects)
      return qc
    }}
  />
)

export const Loading: Story = () => (
  <SidebarWithClient
    createClient={() => new QueryClient({ defaultOptions: { queries: { retry: false, enabled: false } } })}
  />
)
