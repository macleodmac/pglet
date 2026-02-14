import type { Meta, StoryObj } from '@storybook/react'
import { Pagination } from './Pagination'
import { useState } from 'react'

const meta: Meta<typeof Pagination> = {
  title: 'Results/Pagination',
  component: Pagination,
}

export default meta
type Story = StoryObj<typeof Pagination>

export const FirstPage: Story = {
  args: {
    page: 0,
    pageSize: 100,
    totalCount: 1542,
    onPageChange: () => {},
  },
}

export const MiddlePage: Story = {
  args: {
    page: 5,
    pageSize: 100,
    totalCount: 1542,
    onPageChange: () => {},
  },
}

export const LastPage: Story = {
  args: {
    page: 15,
    pageSize: 100,
    totalCount: 1542,
    onPageChange: () => {},
  },
}

export const SinglePage: Story = {
  args: {
    page: 0,
    pageSize: 100,
    totalCount: 50,
    onPageChange: () => {},
  },
}

export const Interactive: Story = {
  render: () => {
    const [page, setPage] = useState(0)
    return (
      <Pagination
        page={page}
        pageSize={100}
        totalCount={2500}
        onPageChange={setPage}
      />
    )
  },
}
