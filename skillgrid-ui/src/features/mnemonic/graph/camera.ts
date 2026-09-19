import type { CameraState, Extent } from 'sigma/types'

// BBox is the shape of sigma.getBBox(): { x: [min,max], y: [min,max] }.
export interface BBox {
  x: Extent
  y: Extent
}

// Dimensions is the shape of sigma.getDimensions(): { width, height }.
export interface Dimensions {
  width: number
  height: number
}

// fitCameraToBBox computes the camera state that frames a bounding box inside
// the viewport, leaving `padding` (fraction of each dimension) as margin. It's
// pure (no Sigma import) so the fit math is unit-testable.
//
// ratio = max(vw*(1-2p)/w, vh*(1-2p)/h) — the zoom level that makes BOTH the
// box width and height fit (the larger of the two ratios fits the tighter
// axis). x/y = the box center. angle = 0 (no rotation).
//
// A zero-area bbox (single node / empty graph) falls back to a 1x1 box so the
// ratio is finite instead of Infinity.
export function fitCameraToBBox(
  box: BBox,
  dims: Dimensions,
  padding = 0.1,
): CameraState {
  const w = box.x[1] - box.x[0] || 1
  const h = box.y[1] - box.y[0] || 1
  const usableW = dims.width * (1 - 2 * padding)
  const usableH = dims.height * (1 - 2 * padding)
  const ratio = Math.max(usableW / w, usableH / h)
  return {
    x: (box.x[0] + box.x[1]) / 2,
    y: (box.y[0] + box.y[1]) / 2,
    angle: 0,
    ratio,
  }
}
