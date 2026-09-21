import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DecisionCard, type Decision } from './DecisionCard'

const pending: Decision = {
  id: 1,
  topicKey: 'interview/demo/layout-1',
  title: 'Layout',
  createdAt: '2026-09-19T10:00:00Z',
  updatedAt: '2026-09-19T10:00:00Z',
  visibility: 'team',
  content: {
    question: 'Which layout?',
    options: [
      { id: 'a', label: 'Single column' },
      { id: 'b', label: 'Two column' },
    ],
    recommended: 'a',
    state: 'pending',
  },
}

const answered: Decision = {
  ...pending,
  id: 2,
  content: {
    ...pending.content,
    state: 'answered',
    answeredOption: 'b',
    answerNote: 'picked two column for density',
  },
}

describe('DecisionCard', () => {
  it('renders options with the recommended one highlighted', () => {
    render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
    expect(screen.getByText('Which layout?')).toBeTruthy()
    expect(screen.getByText('Single column')).toBeTruthy()
    expect(screen.getByText('Two column')).toBeTruthy()
    const rec = screen.getByTestId('decision-option-a')
    expect(rec.className).toContain('recommended')
  })

  it('renders no iframe for a decision without visual', () => {
    const { container } = render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
    expect(container.querySelector('iframe')).toBeNull()
  })

  it('renders the sandboxed iframe for a decision with visual', () => {
    const withVisual: Decision = {
      ...pending,
      content: { ...pending.content, visual: 'demo/a.html' },
    }
    const { container } = render(<DecisionCard decision={withVisual} onAnswer={vi.fn()} />)
    const iframe = container.querySelector('iframe')
    expect(iframe).not.toBeNull()
    expect(iframe?.getAttribute('src')).toBe('/prototype/demo/a.html')
    expect(iframe?.getAttribute('sandbox')).toBe('allow-scripts')
    expect(iframe?.getAttribute('sandbox')).not.toContain('allow-same-origin')
  })

  it('posts the chosen option and note on submit', () => {
    const onAnswer = vi.fn()
    render(<DecisionCard decision={pending} onAnswer={onAnswer} />)
    fireEvent.change(screen.getByPlaceholderText(/optional note/i), {
      target: { value: 'prefer a' },
    })
    fireEvent.click(screen.getByTestId('decision-submit'))
    expect(onAnswer).toHaveBeenCalledWith(1, { optionId: 'a', note: 'prefer a' })
  })

  it('re-submits a different option on an answered decision', () => {
    const onAnswer = vi.fn()
    render(<DecisionCard decision={answered} onAnswer={onAnswer} />)
    fireEvent.click(screen.getByTestId('decision-submit'))
    expect(onAnswer).toHaveBeenCalledWith(2, { optionId: 'b', note: '' })
  })
})
