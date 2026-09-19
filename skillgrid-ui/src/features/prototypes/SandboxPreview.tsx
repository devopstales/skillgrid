// DeviceViewport — the desktop/tablet/mobile frame widths for the sandboxed
// iframe preview.
export const DEVICE_WIDTHS = {
  desktop: 1280,
  tablet: 768,
  mobile: 390,
} as const

export type Device = keyof typeof DEVICE_WIDTHS

// SandboxPreview — a sandboxed iframe (sandbox attr, no allow-same-origin) that
// renders a prototype's HTML via srcDoc, inside a device-width frame. The
// sandbox attr is the key safety property: the prototype's scripts run without
// access to the parent's DOM/storage.
export function SandboxPreview({
  html,
  device,
}: {
  html: string
  device: Device
}) {
  const width = DEVICE_WIDTHS[device]
  return (
    <div
      className="mx-auto overflow-hidden rounded-lg border border-edge bg-white shadow-lg"
      style={{ width, maxWidth: '100%' }}
    >
      <iframe
        title="prototype preview"
        srcDoc={html}
        sandbox="allow-scripts"
        className="h-[640px] w-full border-0"
      />
    </div>
  )
}
