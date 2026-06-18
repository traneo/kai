import { PipelineBuilder } from './PipelineBuilder'

interface Props {
  onCreated: (runId: string) => void
}

export function NewPipeline({ onCreated }: Props) {
  return (
    <div>
      <div style={{ marginBottom: 16, padding: '0 4px' }}>
        <h2 style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 4 }}>Create Pipeline</h2>
        <p className="muted">Define your pipeline visually — add steps, configure gates, and set policies.</p>
      </div>
      <PipelineBuilder onCreated={onCreated} />
    </div>
  )
}
