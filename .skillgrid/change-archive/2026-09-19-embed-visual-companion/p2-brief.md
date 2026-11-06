### TICKET-03 — RED: frontend /decisions view acceptance tests (from tasks.md)
### TICKET-03 — RED: frontend `/decisions` view acceptance tests
- **Tracker ID:** TASK-009

- **Scope:** Write the failing React acceptance tests (no components yet) for `/decisions`: a list renders the pending decision inbox (title + body + created), clicking a card opens the detail with a textarea + submit, submitting POSTs to `POST /mnemonic/decisions/{id}/answer` and on success shows `state=answered` + the stored answer, a decision without `visual` renders no iframe, a decision with `visual` renders the sandboxed iframe at `/prototype/{visual}`.
- **Acceptance:** `pnpm --filter skillgrid-ui test` (or the project's vitest runner) fails on the decision view tests (route/component not yet present) with the expected assertions; tests compile.
- **SATISFIES:** Scenario "dashboard shows the pending decision inbox" + Scenario "user answers in the dashboard, it round-trips" + Scenario "visual companion renders a throwaway prototype" (RED state).
- **Files:** `skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx`, `DecisionCard.test.tsx` (new).
- **Size:** ~200 (S)
- **Blocks:** TICKET-04, TICKET-05
- **Blocked by:** none
- **Fails-when:** the UI test runner exits non-zero on the decision view tests (expected RED) but the UI package compiles.

### TICKET-04 — /decisions dashboard view (from tasks.md)
### TICKET-04 — `/decisions` dashboard view (inbox + answer round-trip)
- **Tracker ID:** TASK-010

- **Scope:** Build the `/decisions` route + `DecisionsPage` (left inbox list of pending decisions, right detail), `DecisionCard` (title/body/created + answer textarea + submit → `POST .../answer`, on success show `state=answered` + stored answer, idempotent re-answer is a no-op), `api.ts` (fetch `GET /mnemonic/decisions`, POST answer). Add the `/decisions` route to `app.tsx` and a nav entry to `AppLayout.tsx`.
- **Acceptance:** TICKET-03 tests now pass; in the live dashboard, the pending decision appears in the inbox, the user types an answer and submits, the card flips to `state=answered` and shows the stored answer, a second submit is a no-op, and the decision no longer appears in the pending list on reload.
- **SATISFIES:** Scenario "dashboard shows the pending decision inbox" + Scenario "user answers in the dashboard, it round-trips" (GREEN).
- **Files:** `skillgrid-ui/src/features/decisions/DecisionsPage.tsx`, `DecisionCard.tsx`, `api.ts` (new), `skillgrid-ui/src/app.tsx` (route), `skillgrid-ui/src/components/layout/AppLayout.tsx` (nav).
- **Size:** ~350 (M)
- **Blocks:** TICKET-05
- **Blocked by:** TICKET-02, TICKET-03
- **Fails-when:** the UI test runner exits non-zero on the decision view tests.

### Blueprint Task 3 (full)
### Task 3: React `/decisions` view (decision inbox + answer round-trip)

**Files:**
- Create: `skillgrid-ui/src/features/decisions/api.ts`
- Create: `skillgrid-ui/src/features/decisions/DecisionsPage.tsx`
- Create: `skillgrid-ui/src/features/decisions/DecisionCard.tsx`
- Modify: `skillgrid-ui/src/app.tsx` (add `decisionsRoute`)
- Modify: `skillgrid-ui/src/components/layout/AppLayout.tsx` (add nav entry)

**Interfaces:**
- Consumes: `GET /mnemonic/decisions?project=…&state=…`, `POST /mnemonic/decisions/{id}/answer?project=…`; `currentProjectName` from `../../lib/projects`.
- Produces: `DecisionsPage` (default-exported route component); `decisionsApi` with `fetchDecisions(project, state)`, `answerDecision(project, id, {optionId, note})`.

**SATISFIES:** scenarios `decision-view-renders` + `decision-view-roundtrip`.

- [ ] **Step 1: Write the failing test**

```tsx
// skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DecisionCard } from './DecisionCard'

const d = {
  id: 1, topicKey: 'interview/demo/layout-1', title: 'Layout',
  createdAt: '', updatedAt: '', visibility: 'team', parseError: false,
  content: {
    question: 'Which layout?',
    options: [{ id: 'a', label: 'Single column' }, { id: 'b', label: 'Two column' }],
    recommended: 'a', state: 'pending',
  },
}

describe('DecisionCard', () => {
  it('renders options with the recommended one highlighted', () => {
    render(<DecisionCard decision={d as any} onAnswer={vi.fn()} />)
    expect(screen.getByText('Which layout?')).toBeTruthy()
    const rec = screen.getByTestId('decision-option-a')
    expect(rec.className).toContain('recommended')
  })
  it('posts the chosen option on click', () => {
    const onAnswer = vi.fn()
    render(<DecisionCard decision={d as any} onAnswer={onAnswer} />)
    fireEvent.click(screen.getByTestId('decision-option-b'))
    expect(onAnswer).toHaveBeenCalledWith(1, { optionId: 'b', note: '' })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-ui && npm test -- DecisionCard`
Expected: FAIL — module not found.

- [ ] **Step 3: Write minimal implementation**

```tsx
// DecisionCard.tsx
import { useState } from 'react'

export interface DecisionOption { id: string; label: string; rationale?: string }
export interface Decision {
  id: number; topicKey: string; title: string; createdAt: string; updatedAt: string
  visibility: string; parseError?: boolean
  content: { question: string; options: DecisionOption[]; recommended?: string; state: string; answeredOption?: string; answerNote?: string }
}

export function DecisionCard({ decision, onAnswer }: { decision: Decision; onAnswer: (id: number, a: { optionId: string; note: string }) => void }) {
  const [note, setNote] = useState('')
  const answered = decision.content.state === 'answered'
  if (decision.parseError) {
    return <div className="rounded-md border border-warn bg-warn-soft p-3 text-sm">Unreadable decision: {decision.topicKey}</div>
  }
  return (
    <div className="rounded-md border border-edge bg-panel p-4">
      <div className="mb-1 text-xs uppercase tracking-wide text-ink-3">{decision.topicKey}</div>
      <h3 className="mb-3 font-semibold">{decision.content.question}</h3>
      <div className="flex flex-col gap-2">
        {decision.content.options.map((o) => {
          const isRec = o.id === decision.content.recommended
          const isPicked = answered && o.id === decision.content.answeredOption
          return (
            <button
              key={o.id}
              data-testid={`decision-option-${o.id}`}
              disabled={answered}
              onClick={() => onAnswer(decision.id, { optionId: o.id, note })}
              className={[
                'rounded border p-2 text-left text-sm transition',
                isRec ? 'border-accent bg-accent-soft recommended' : 'border-edge hover:bg-background',
                isPicked ? 'border-ok bg-ok-soft' : '',
              ].join(' ')}
            >
              <span className="font-medium">{o.label}</span>
              {isRec && !answered && <span className="ml-2 text-xs text-accent">recommended</span>}
              {isPicked && <span className="ml-2 text-xs text-ok">your choice</span>}
              {o.rationale && <div className="mt-1 text-xs text-ink-2">{o.rationale}</div>}
            </button>
          )
        })}
      </div>
      {!answered && (
        <input
          className="mt-3 w-full rounded border border-edge bg-background px-2 py-1 text-sm"
          placeholder="Optional note for the agent…"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      )}
      {answered && decision.content.answerNote && (
        <div className="mt-2 text-xs text-ink-2">Note: {decision.content.answerNote}</div>
      )}
    </div>
  )
}
```

```tsx
// DecisionsPage.tsx
import { useCallback, useEffect, useState } from 'react'
import { DecisionCard, type Decision } from './DecisionCard'
import * as api from './api'

export function DecisionsPage() {
  const [tab, setTab] = useState<'pending' | 'answered' | 'all'>('pending')
  const [rows, setRows] = useState<Decision[]>([])
  const [err, setErr] = useState<string | null>(null)

  const load = useCallback(async () => {
    try { setRows(await api.fetchDecisions(tab === 'all' ? '' : tab)) }
    catch (e) { setErr(String(e)) }
  }, [tab])

  useEffect(() => { void load(); const t = setInterval(load, 4000); return () => clearInterval(t) }, [load])

  const onAnswer = useCallback(async (id: number, a: { optionId: string; note: string }) => {
    await api.answerDecision(id, a); void load()
  }, [load])

  return (
    <div className="flex h-full flex-col gap-3 overflow-y-auto p-6">
      <div className="flex gap-1 text-sm">
        {(['pending', 'answered', 'all'] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={'rounded px-3 py-1 ' + (tab === t ? 'bg-accent-soft text-accent' : 'text-ink-2 hover:bg-background')}>{t}</button>
        ))}
      </div>
      {err && <div className="text-error">{err}</div>}
      {rows.length === 0 && <div className="text-ink-3">No {tab} decisions.</div>}
      <div className="flex flex-col gap-3">
        {rows.map((d) => <DecisionCard key={d.id} decision={d} onAnswer={onAnswer} />)}
      </div>
    </div>
  )
}
```

```ts
// api.ts
import { currentProjectName } from '../../lib/projects'
import type { Decision } from './DecisionCard'

let project: string | null = null
async function resolveProject(): Promise<string> {
  if (project) return project
  project = await currentProjectName()
  return project
}

export async function fetchDecisions(state: string): Promise<Decision[]> {
  const p = await resolveProject()
  const q = new URLSearchParams({ project: p }); if (state) q.set('state', state)
  const res = await fetch(`/mnemonic/decisions?${q}`)
  if (!res.ok) throw new Error(`decisions ${res.status}`)
  return res.json()
}

export async function answerDecision(id: number, a: { optionId: string; note: string }): Promise<void> {
  const p = await resolveProject()
  const res = await fetch(`/mnemonic/decisions/${id}/answer?project=${encodeURIComponent(p)}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(a),
  })
  if (!res.ok) throw new Error(`answer ${res.status}`)
}
```

Wire route + nav (mirror `activityRoute` / `STANDALONE_ITEMS`):

```tsx
// app.tsx
const decisionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/decisions',
  component: lazyPage(() => import('./features/decisions/DecisionsPage').then((m) => ({ default: m.DecisionsPage }))),
})
// add decisionsRoute to routeTree.children
// AppLayout.tsx STANDALONE_ITEMS: { to: '/decisions', label: 'Decisions' },
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-ui && npm test -- DecisionCard && npm run build:check`
Expected: PASS (tests + bundle budget).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-ui/src/features/decisions/ skillgrid-ui/src/app.tsx skillgrid-ui/src/components/layout/AppLayout.tsx
git commit -m "feat(ui): decisions view — decision inbox + answer round-trip (polling)"
```

---

