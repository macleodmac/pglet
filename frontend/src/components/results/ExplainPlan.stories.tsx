import type { Story, StoryDefault } from '@ladle/react'
import { ExplainPlan } from './ExplainPlan'
import { analyzedPlan, estimatePlan } from './explainPlan.fixtures'

export default {
  title: 'Results/ExplainPlan',
} satisfies StoryDefault

const Frame = ({ children }: { children: React.ReactNode }) => <div className="h-[480px]">{children}</div>

export const Analyzed: Story = () => (
  <Frame>
    <ExplainPlan rows={[[analyzedPlan]]} durationMs={42} />
  </Frame>
)

export const Estimate: Story = () => (
  <Frame>
    <ExplainPlan rows={[[estimatePlan]]} durationMs={3} />
  </Frame>
)

export const TextFormat: Story = () => (
  <Frame>
    <ExplainPlan
      rows={[
        ['Hash Join  (cost=1541.00..13985.00 rows=50000 width=55)'],
        ['  Hash Cond: (b.id = o.id)'],
        ['  ->  Seq Scan on big b  (cost=0.00..11056.00 rows=600000 width=37)'],
        ['  ->  Hash  (cost=791.00..791.00 rows=50000 width=18)'],
        ['        ->  Seq Scan on orders o  (cost=0.00..791.00 rows=50000 width=18)'],
      ]}
      durationMs={1}
    />
  </Frame>
)
