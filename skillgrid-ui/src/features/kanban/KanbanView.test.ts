import { describe, it, expect } from 'vitest'
import { statusForMove } from './KanbanView'

describe('statusForMove', () => {
  it('maps the ready column to ready-for-agent for backlogmd', () => {
    expect(statusForMove('backlogmd', 'ready')).toBe('ready-for-agent')
  })

  it('maps todo to needs-triage, blocked to blocked, done to done (backlogmd)', () => {
    expect(statusForMove('backlogmd', 'todo')).toBe('needs-triage')
    expect(statusForMove('backlogmd', 'blocked')).toBe('blocked')
    expect(statusForMove('backlogmd', 'done')).toBe('done')
    expect(statusForMove('backlogmd', 'in_progress')).toBe('ready-for-agent')
  })

  it('maps cli-backed providers to closed/open', () => {
    expect(statusForMove('github', 'done')).toBe('closed')
    expect(statusForMove('github', 'ready')).toBe('open')
    expect(statusForMove('jira', 'todo')).toBe('open')
  })
})
