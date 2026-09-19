import { useSigma, useCamera } from '@react-sigma/core'
import { fitCameraToBBox } from '../camera'

// ViewControls renders Zoom In / Zoom Out / Fit-to-Screen. It MUST be a child of
// SigmaContainer because useSigma/useCamera read the Sigma context (the camera
// isn't reachable from outside the provider).
export function ViewControls() {
  const sigma = useSigma()
  const { zoomIn, zoomOut } = useCamera()

  const fitToScreen = () => {
    const cam = fitCameraToBBox(sigma.getBBox(), sigma.getDimensions(), 0.1)
    sigma.getCamera().animate(cam, { duration: 400, easing: 'quadraticInOut' })
  }

  const btn =
    'flex h-7 w-7 items-center justify-center rounded-md border border-slate-700 bg-slate-900/80 text-sm text-slate-200 hover:bg-slate-800'

  return (
    <div className="absolute top-3 left-1/2 z-10 flex -translate-x-1/2 gap-1">
      <button type="button" onClick={() => zoomIn()} className={btn} title="Zoom in" aria-label="Zoom in">
        +
      </button>
      <button type="button" onClick={() => zoomOut()} className={btn} title="Zoom out" aria-label="Zoom out">
        −
      </button>
      <button
        type="button"
        onClick={fitToScreen}
        className={`${btn} w-auto px-2 text-xs`}
        title="Fit to screen"
        aria-label="Fit to screen"
      >
        Fit
      </button>
    </div>
  )
}
