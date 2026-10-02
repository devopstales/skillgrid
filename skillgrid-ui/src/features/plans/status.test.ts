import { describe, it, expect } from 'vitest'
import { chipKind } from './status'

describe('chipKind', () => {
  it('maps shipped/done statuses to pass', () => {
    expect(chipKind('done')).toBe('pass')
    expect(chipKind('COMPLETE')).toBe('pass')
    expect(chipKind('shipped')).toBe('pass')
    expect(chipKind('archived')).toBe('pass')
  })

  it('maps live work to active', () => {
    expect(chipKind('in-progress')).toBe('active')
    expect(chipKind('in_progress')).toBe('active')
    expect(chipKind('executing')).toBe('active')
    expect(chipKind('approved')).toBe('active')
  })

  it('maps failures to blocked, even when another keyword is present', () => {
    expect(chipKind('blocked')).toBe('blocked')
    expect(chipKind('qa-failed')).toBe('blocked')
  })

  it('falls back to pending for unknown or empty statuses', () => {
    expect(chipKind('PENDING')).toBe('pending')
    expect(chipKind('draft')).toBe('pending')
    expect(chipKind('')).toBe('pending')
    expect(chipKind(undefined)).toBe('pending')
  })
})
