import { describe, it, expect } from 'vitest'
import { fitCameraToBBox } from './camera'

// ratio = max(vw*(1-2p)/w, vh*(1-2p)/h). For p=0.1: vw*0.8, vh*0.8.
describe('fitCameraToBBox', () => {
  it('centers on the bbox and fits the tighter axis', () => {
    // box 100 wide x 50 tall, viewport 800x600, padding 0.1
    // usable 640x480; w=100 h=50
    // ratio = max(640/100, 480/50) = max(6.4, 9.6) = 9.6 (height tighter)
    const cam = fitCameraToBBox(
      { x: [0, 100], y: [0, 50] },
      { width: 800, height: 600 },
      0.1,
    )
    expect(cam.x).toBe(50)
    expect(cam.y).toBe(25)
    expect(cam.angle).toBe(0)
    expect(cam.ratio).toBeCloseTo(9.6, 5)
  })

  it('uses the width ratio when width is the tighter axis', () => {
    // box 50 wide x 100 tall → width tighter: ratio = max(640/50, 480/100)=max(12.8,4.8)=12.8
    const cam = fitCameraToBBox(
      { x: [-25, 25], y: [-50, 50] },
      { width: 800, height: 600 },
      0.1,
    )
    expect(cam.x).toBe(0)
    expect(cam.y).toBe(0)
    expect(cam.ratio).toBeCloseTo(12.8, 5)
  })

  it('handles a zero-area bbox (single node) with a finite ratio', () => {
    // w=h=1 (|| 1), usable 640x480 → ratio = max(640, 480) = 640
    const cam = fitCameraToBBox(
      { x: [10, 10], y: [10, 10] },
      { width: 800, height: 600 },
      0.1,
    )
    expect(cam.x).toBe(10)
    expect(cam.y).toBe(10)
    expect(Number.isFinite(cam.ratio)).toBe(true)
    expect(cam.ratio).toBe(640)
  })

  it('respects the padding fraction (larger padding → smaller zoom)', () => {
    // Compute expected directly from the formula to avoid hand-math drift.
    const vw = 800, vh = 600, p = 0.2, bw = 100, bh = 50
    const expected = Math.max(vw * (1 - 2 * p) / bw, vh * (1 - 2 * p) / bh)
    const cam = fitCameraToBBox(
      { x: [0, 100], y: [0, 50] },
      { width: vw, height: vh },
      p,
    )
    expect(cam.ratio).toBeCloseTo(expected, 5)
    // more padding → smaller ratio (less zoomed in)
    const tight = fitCameraToBBox({ x: [0, 100], y: [0, 50] }, { width: vw, height: vh }, 0.05)
    expect(cam.ratio).toBeLessThan(tight.ratio)
  })

  it('defaults to 0.1 padding when omitted', () => {
    const a = fitCameraToBBox({ x: [0, 100], y: [0, 50] }, { width: 800, height: 600 })
    const b = fitCameraToBBox({ x: [0, 100], y: [0, 50] }, { width: 800, height: 600 }, 0.1)
    expect(a.ratio).toBe(b.ratio)
  })
})
