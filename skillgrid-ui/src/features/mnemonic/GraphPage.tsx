import { useState } from 'react'
import { VectorGraph } from './graph/VectorGraph'
import { ExplorerPanel } from './graph/ExplorerPanel'
import { useMnemonicGraph, type UseMnemonicGraph } from './graph/hooks/useMnemonicGraph'

// GraphPage lays out the explorer (left) + the graph (right) as a flex row.
// It owns the graph data (useMnemonicGraph) so the ExplorerPanel and the
// VectorGraph share one fetch. The explorer is collapsible; agent chips + the
// selected file drive node dimming in the graph (via VectorGraph props).
export function GraphPage() {
  const graphState: UseMnemonicGraph = useMnemonicGraph()
  const { raw } = graphState
  const [collapsed, setCollapsed] = useState(false)
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [activeAgents, setActiveAgents] = useState<Set<string>>(new Set())

  const toggleAgent = (agent: string) => {
    setActiveAgents((prev) => {
      const next = new Set(prev)
      if (next.has(agent)) next.delete(agent)
      else next.add(agent)
      return next
    })
  }

  const nodes = raw?.nodes ?? []

  return (
    <div className="flex h-[calc(100vh-4rem)] w-full">
      <ExplorerPanel
        nodes={nodes}
        collapsed={collapsed}
        onToggleCollapse={() => setCollapsed((c) => !c)}
        onSelectFile={(p) => setSelectedFile((cur) => (cur === p ? null : p))}
        activeAgents={activeAgents}
        onToggleAgent={toggleAgent}
      />
      <div className="flex-1">
        <VectorGraph
          graphState={graphState}
          selectedFile={selectedFile}
          activeAgents={activeAgents}
        />
      </div>
    </div>
  )
}
