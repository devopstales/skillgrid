// Plan → ticket cross-links. Scans plan markdown (ledger steps, briefing,
// tasks) for task references and wraps them in deep links to the tracker.
// Three reference forms are recognized:
//   task-003            → /tracker?task=003
//   #003                → /tracker?task=003
//   .backlog/tasks/003-*.md → /tracker?task=003
// The match is a global regex; linkifyTaskRefs replaces each match with an
// anchor. Repeated refs each become their own anchor (no dedup across
// occurrences — the same id can legitimately appear in multiple steps).

export const TASK_REF_RE = /(?:task-|\.backlog\/tasks\/|#)(\d{3,})/g

export function linkifyTaskRefs(text: string): string {
  return text.replace(
    new RegExp(TASK_REF_RE.source, 'g'),
    (match, id: string) => {
      return `<a href="/tracker?task=${id}" class="text-accent hover:underline">${match}</a>`
    },
  )
}

// linkifyTaskRefsMarkdown is the same scan for text that still goes through a
// markdown renderer (briefing). It emits markdown links so the renderer, not
// raw HTML, produces the anchors.
export function linkifyTaskRefsMarkdown(text: string): string {
  return text.replace(
    new RegExp(TASK_REF_RE.source, 'g'),
    (match, id: string) => `[${match}](/tracker?task=${id})`,
  )
}
