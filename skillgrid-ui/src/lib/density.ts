// Phase 7.4 density modes (comfortable/compact). The preference is persisted in
// localStorage and applied as a data-density attribute on <html>; the CSS
// tokens in index.css respond to it. Kept dependency-free.
const KEY = 'skillgrid.density'

export type Density = 'comfortable' | 'compact'

export function getDensity(): Density {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'comfortable' || v === 'compact') return v
  } catch {
    /* localStorage unavailable */
  }
  return 'comfortable'
}

export function setDensity(d: Density) {
  try {
    localStorage.setItem(KEY, d)
  } catch {
    /* ignore */
  }
  applyDensity(d)
}

export function applyDensity(d: Density) {
  document.documentElement.dataset.density = d
}
