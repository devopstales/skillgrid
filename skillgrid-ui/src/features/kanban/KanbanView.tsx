import { useMemo, useState } from 'react'
import { updateTaskStatus, TrackerError } from './api'
import { useTaskDrawer, useTrackerBoard } from './hooks'
import {
  EMPTY_FILTERS,
  PROVIDERS,
  PROVIDER_LABEL,
  type BoardColumn,
  type ProviderName,
} from './types'
import { BoardView } from './BoardView'
import { ListView } from './ListView'
import { FilterBar, applyFilters } from './FilterBar'
import { TaskDetail } from './TaskDetail'

// mapBoardToStatus picks a provider-native status to POST when a card is
// dropped into a column. Only Backlog.md has rich statuses; for CLI-backed
// providers we map done→closed/close and everything else→open/reopen so the
// round-trip always hits a supported transition (501 otherwise).
function statusForMove(provider: ProviderName, to: BoardColumn): string {
  if (provider === 'backlogmd') {
    switch (to) {
      case 'todo':
        return 'needs-triage'
      case 'in_progress':
        return 'ready-for-agent'
      case 'blocked':
        return 'blocked'
      case 'done':
        return 'done'
    }
  }
  return to === 'done' ? 'closed' : 'open'
}

export function KanbanView() {
  const params = new URLSearchParams(window.location.search)
  const initialProvider = (params.get('provider') as ProviderName) ?? 'backlogmd'
  const [provider, setProvider] = useState<ProviderName>(
    PROVIDERS.includes(initialProvider) ? initialProvider : 'backlogmd',
  )
  const { state, live, reload } = useTrackerBoard(provider)
  const { selected, open, close } = useTaskDrawer()
  const [view, setView] = useState<'board' | 'list'>('board')
  const [filters, setFilters] = useState(EMPTY_FILTERS)
  const [moveError, setMoveError] = useState('')

  const tasks = useMemo(
    () => (state.status === 'ready' ? state.tasks : []),
    [state],
  )
  const visible = useMemo(() => applyFilters(tasks, filters), [tasks, filters])

  function switchProvider(p: ProviderName) {
    setProvider(p)
    const url = new URL(window.location.href)
    url.searchParams.set('provider', p)
    window.history.replaceState(null, '', url.toString())
  }

  async function onMove(id: string, to: BoardColumn) {
    const task = tasks.find((t) => t.id === id)
    if (!task || task.board === to) return
    setMoveError('')
    const status = statusForMove(provider, to)
    try {
      await updateTaskStatus(id, status, provider)
      await reload()
    } catch (err) {
      if (err instanceof TrackerError && err.status === 401) {
        setMoveError('Write needs a token — set it in Settings.')
      } else if (err instanceof TrackerError && err.status === 501) {
        setMoveError(`Provider can't move to "${status}": ${err.message}`)
      } else {
        setMoveError(err instanceof Error ? err.message : 'move failed')
      }
    }
  }

  const degraded = state.status === 'degraded'
  const boardDisabled = degraded || state.status === 'error'

  return (
    <div className="flex h-full flex-col">
      {/* Provider tabs + view toggle + live indicator */}
      <div className="flex items-center gap-1 border-b border-edge px-4 pt-3">
        {PROVIDERS.map((p) => (
          <button
            key={p}
            type="button"
            onClick={() => switchProvider(p)}
            className={[
              'rounded-t-md px-3 py-1.5 text-sm transition-colors',
              provider === p
                ? 'bg-card text-zinc-100'
                : 'text-zinc-500 hover:text-zinc-300',
            ].join(' ')}
          >
            {PROVIDER_LABEL[p]}
          </button>
        ))}
        <div className="ml-auto flex items-center gap-3">
          {live && (
            <span className="flex items-center gap-1.5 text-[11px] text-emerald-400">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
              live
            </span>
          )}
          <div className="flex overflow-hidden rounded-md border border-edge">
            {(['board', 'list'] as const).map((v) => (
              <button
                key={v}
                type="button"
                onClick={() => setView(v)}
                className={[
                  'px-2.5 py-1 text-xs capitalize',
                  view === v
                    ? 'bg-edge/60 text-zinc-100'
                    : 'text-zinc-500 hover:text-zinc-300',
                ].join(' ')}
              >
                {v}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Filter bar (hidden when degraded) */}
      {!degraded && state.status !== 'error' && (
        <FilterBar filters={filters} onChange={setFilters} tasks={tasks} />
      )}

      {moveError && (
        <div className="border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-xs text-amber-300">
          {moveError}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto">
        {state.status === 'loading' && (
          <div className="flex h-full items-center justify-center text-sm text-zinc-500">
            Loading {PROVIDER_LABEL[provider]}…
          </div>
        )}

        {degraded && (
          <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center">
            <span className="text-sm font-medium text-zinc-300">
              {PROVIDER_LABEL[provider]} not connected
            </span>
            <span className="max-w-md text-xs text-zinc-500">
              {state.status === 'degraded' ? state.reason : ''}
            </span>
            <span className="mt-2 text-[11px] text-zinc-600">
              Install &amp; authenticate the provider CLI, then retry.
            </span>
            <button
              type="button"
              onClick={() => void reload()}
              className="mt-3 rounded-md border border-edge bg-card px-3 py-1.5 text-xs text-zinc-300 hover:border-accent/50"
            >
              Retry
            </button>
          </div>
        )}

        {state.status === 'error' && (
          <div className="flex h-full items-center justify-center p-8">
            <span className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-300">
              {state.message}
            </span>
          </div>
        )}

        {state.status === 'ready' &&
          (view === 'board' ? (
            <BoardView
              tasks={visible}
              onOpenTask={open}
              onMove={onMove}
              disabled={boardDisabled}
            />
          ) : (
            <ListView tasks={visible} onOpenTask={open} />
          ))}
      </div>

      <TaskDetail task={selected} provider={provider} onClose={close} />
    </div>
  )
}
