import { describe, it, expect } from 'vitest'
import { linkifyTaskRefs, linkifyTaskRefsMarkdown, TASK_REF_RE } from './taskLinks'

function matchId(text: string): string | undefined {
  const m = new RegExp(TASK_REF_RE.source, 'g').exec(text)
  return m?.[1]
}

describe('TASK_REF_RE', () => {
  it('matches task-NNN form', () => {
    expect(matchId('see task-003 for details')).toBe('003')
  })

  it('matches #NNN form', () => {
    expect(matchId('linked to #003')).toBe('003')
  })

  it('matches .backlog/tasks/NNN-*.md path form', () => {
    expect(matchId('read .backlog/tasks/003-fix-login.md')).toBe('003')
  })

  it('does not match bare numbers without a prefix', () => {
    expect(matchId('version 3 is out')).toBeUndefined()
  })
})

describe('linkifyTaskRefs', () => {
  it('wraps task-NNN in a tracker deep link', () => {
    const out = linkifyTaskRefs('do task-003 then task-007')
    expect(out).toContain('href="/tracker?task=003"')
    expect(out).toContain('href="/tracker?task=007"')
  })

  it('wraps #NNN in a tracker deep link', () => {
    expect(linkifyTaskRefs('see #012')).toContain('href="/tracker?task=012"')
  })

  it('emits a markdown link for text that a markdown renderer will display', () => {
    expect(linkifyTaskRefsMarkdown('See #012 for the design.')).toBe(
      'See [#012](/tracker?task=012) for the design.',
    )
  })

  it('keeps surrounding text intact', () => {
    const out = linkifyTaskRefs('do task-003 now')
    expect(out).toContain('do ')
    expect(out).toContain(' now')
  })

  it('returns the text unchanged when no refs are present', () => {
    expect(linkifyTaskRefs('no refs here')).toBe('no refs here')
  })

  it('deduplicates repeated refs', () => {
    const out = linkifyTaskRefs('task-003 and task-003 again')
    const count = out.split('task-003').length - 1
    expect(count).toBe(2)
  })
})
