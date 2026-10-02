import { useEffect, useState } from 'react'
import { apiGet } from '../../lib/api'
import {
  EmptyState,
  ErrorState,
  LoadingState,
  PageHeader,
  SectionTitle,
} from '../../components/ui/Badges'

interface CompactionBlock {
  timestamp?: string
  summary?: string
  observations?: unknown[]
}

interface CompactionResponse {
  blocks?: CompactionBlock[]
}

interface ContextResponse {
  session_id?: string
  project?: string
  generated_at?: string
  blocks?: unknown[]
  observations?: unknown[]
}

export function CompactionPage() {
  const [compaction, setCompaction] = useState<CompactionResponse | null>(null)
  const [context, setContext] = useState<ContextResponse | null>(null)
  const [compactionError, setCompactionError] = useState<string | null>(null)
  const [contextError, setContextError] = useState<string | null>(null)

  useEffect(() => {
    apiGet<CompactionResponse>('/context/compaction')
      .then(setCompaction)
      .catch((err: Error) => setCompactionError(err.message))
    apiGet<ContextResponse>('/context')
      .then(setContext)
      .catch((err: Error) => setContextError(err.message))
  }, [])

  if (!compaction && !context && !compactionError && !contextError) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  return (
    <div className="p-6">
      <PageHeader title="Compaction" subtitle="Session context injection and compaction state" />
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="kpi-card">
          <SectionTitle>Compaction Status</SectionTitle>
          {compactionError ? (
            <ErrorState error={compactionError} />
          ) : compaction?.blocks && compaction.blocks.length > 0 ? (
            <div className="space-y-3">
              {compaction.blocks.map((b, i) => (
                <div key={i} className="rounded border border-edge p-3">
                  <div className="mb-1 text-xs font-semibold text-ink">Block {i + 1}</div>
                  {b.timestamp && (
                    <div className="mb-1 font-mono text-[10px] text-ink-4">{b.timestamp}</div>
                  )}
                  {b.summary && (
                    <div className="whitespace-pre-wrap text-xs text-ink-3">
                      {b.summary.slice(0, 300)}
                    </div>
                  )}
                  {b.observations && b.observations.length > 0 && (
                    <div className="mt-1 text-[10px] text-ink-4">
                      {b.observations.length} observations
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <EmptyState message="No compaction blocks" />
          )}
        </div>
        <div className="kpi-card">
          <SectionTitle>Context</SectionTitle>
          {contextError ? (
            <ErrorState error={contextError} />
          ) : context ? (
            <div className="space-y-2 text-sm">
              {context.session_id && (
                <div className="flex justify-between gap-2">
                  <span className="text-ink-4">Session</span>
                  <span className="max-w-[200px] truncate font-mono text-xs text-ink">
                    {context.session_id}
                  </span>
                </div>
              )}
              {context.project && (
                <div className="flex justify-between">
                  <span className="text-ink-4">Project</span>
                  <span className="font-mono text-xs">{context.project}</span>
                </div>
              )}
              {context.generated_at && (
                <div className="flex justify-between">
                  <span className="text-ink-4">Generated</span>
                  <span className="font-mono text-xs text-ink-4">
                    {context.generated_at.slice(0, 19).replace('T', ' ')}
                  </span>
                </div>
              )}
              {context.blocks && (
                <div className="flex justify-between">
                  <span className="text-ink-4">Context Blocks</span>
                  <span className="font-mono text-xs">{context.blocks.length}</span>
                </div>
              )}
              {context.observations && (
                <div className="flex justify-between">
                  <span className="text-ink-4">Observations</span>
                  <span className="font-mono text-xs">{context.observations.length}</span>
                </div>
              )}
            </div>
          ) : (
            <EmptyState message="No context data" />
          )}
        </div>
      </div>
    </div>
  )
}
