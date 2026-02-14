import type { Meta, StoryObj } from '@storybook/react'
import { JsonbViewer } from './JsonbViewer'

const meta: Meta<typeof JsonbViewer> = {
  title: 'Results/JsonbViewer',
  component: JsonbViewer,
}

export default meta
type Story = StoryObj<typeof JsonbViewer>

export const SimpleObject: Story = {
  args: {
    data: {
      name: 'Alice',
      age: 30,
      active: true,
      email: null,
    },
  },
}

export const NestedObject: Story = {
  args: {
    data: {
      user: {
        id: 1,
        profile: {
          name: 'Bob',
          address: {
            city: 'San Francisco',
            state: 'CA',
            zip: '94105',
          },
        },
      },
      permissions: ['read', 'write', 'admin'],
      metadata: {
        created: '2024-01-15T10:30:00Z',
        tags: ['premium', 'beta-tester'],
      },
    },
  },
}

export const ArrayData: Story = {
  args: {
    data: [
      { id: 1, name: 'Item 1', price: 9.99 },
      { id: 2, name: 'Item 2', price: 19.99 },
      { id: 3, name: 'Item 3', price: 29.99 },
    ],
  },
}

export const MixedTypes: Story = {
  args: {
    data: {
      string: 'hello world',
      number: 42,
      float: 3.14,
      boolean_true: true,
      boolean_false: false,
      null_value: null,
      empty_array: [],
      empty_object: {},
      nested_array: [1, 'two', true, null, [5, 6]],
    },
  },
}

const largePayloadData = {
  users: Array.from({ length: 20 }, (_, i) => ({
    id: i + 1,
    name: `User ${i + 1}`,
    email: `user${i + 1}@example.com`,
    roles: ['user', ...(i % 3 === 0 ? ['admin'] : [])],
    settings: {
      theme: i % 2 === 0 ? 'dark' : 'light',
      notifications: i % 4 !== 0,
    },
  })),
}

export const LargePayload: Story = {
  args: {
    data: largePayloadData,
    initialExpanded: 1,
  },
}

export const CollapsedByDefault: Story = {
  args: {
    data: { a: { b: { c: { d: 'deep' } } } },
    initialExpanded: 0,
  },
}
