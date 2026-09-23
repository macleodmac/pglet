import type { Story, StoryDefault } from '@ladle/react'
import { Pagination } from './Pagination'
import { useState } from 'react'

export default {
  title: 'Results/Pagination',
} satisfies StoryDefault

const noop = () => {}

export const FirstPage: Story = () => <Pagination page={0} pageSize={100} totalCount={1542} onPageChange={noop} />

export const MiddlePage: Story = () => <Pagination page={5} pageSize={100} totalCount={1542} onPageChange={noop} />

export const LastPage: Story = () => <Pagination page={15} pageSize={100} totalCount={1542} onPageChange={noop} />

export const SinglePage: Story = () => <Pagination page={0} pageSize={100} totalCount={50} onPageChange={noop} />

export const Interactive: Story = () => {
  const [page, setPage] = useState(0)
  return <Pagination page={page} pageSize={100} totalCount={2500} onPageChange={setPage} />
}
