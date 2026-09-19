import { describe, it, expect } from 'vitest'
import { buildPathTree, agentForPath } from './explorer'

describe('buildPathTree', () => {
  it('builds a nested tree from paths', () => {
    const root = buildPathTree(['a/b/c.ts', 'a/b/d.ts', 'e.ts'])
    const a = root.children!.find((c) => c.name === 'a')!
    const b = a.children!.find((c) => c.name === 'b')!
    expect(b.children!.map((c) => c.file).sort()).toEqual(['a/b/c.ts', 'a/b/d.ts'])
    const e = root.children!.find((c) => c.name === 'e.ts')!
    expect(e.file).toBe('e.ts')
    expect(e.children).toBeUndefined()
  })

  it('dedupes duplicate paths', () => {
    const root = buildPathTree(['a/b/c.ts', 'a/b/c.ts', 'a/b/c.ts'])
    const a = root.children!.find((c) => c.name === 'a')!
    const b = a.children!.find((c) => c.name === 'b')!
    expect(b.children!.length).toBe(1)
  })

  it('sorts directories before files, both alphabetical', () => {
    const root = buildPathTree(['z.ts', 'a.ts', 'dir/file.ts'])
    // top level: dir (directory) first, then a.ts, z.ts
    expect(root.children!.map((c) => c.name)).toEqual(['dir', 'a.ts', 'z.ts'])
  })

  it('caps depth by folding overflow into the last segment', () => {
    const root = buildPathTree(['a/b/c/d/e/f/g/h/i.ts'], 3)
    // depth 3: a → b → (c/d/e/f/g/h/i folded into one leaf name)
    const a = root.children!.find((c) => c.name === 'a')!
    const b = a.children!.find((c) => c.name === 'b')!
    expect(b.children!.length).toBe(1)
    expect(b.children![0].file).toBe('a/b/c/d/e/f/g/h/i.ts')
  })

  it('handles an empty list', () => {
    const root = buildPathTree([])
    expect(root.children!.length).toBe(0)
  })

  it('strips a leading slash', () => {
    const root = buildPathTree(['/a/b.ts'])
    const a = root.children!.find((c) => c.name === 'a')!
    expect(a.children![0].file).toBe('a/b.ts')
  })
})

describe('agentForPath', () => {
  it('maps .cursor/ paths to cursor', () => {
    expect(agentForPath('.cursor/rules/foo.md')).toBe('cursor')
    expect(agentForPath('a/.cursor/settings.json')).toBe('cursor')
    expect(agentForPath('my.cursorrules')).toBe('cursor')
  })

  it('maps kilo paths to kilo', () => {
    expect(agentForPath('plugins/kilo/mnemonic.ts')).toBe('kilo')
    expect(agentForPath('kilo/config.yaml')).toBe('kilo')
  })

  it('maps .opencode/ paths to opencode', () => {
    expect(agentForPath('.opencode/agents/x.md')).toBe('opencode')
    expect(agentForPath('plugins/opencode/tool.ts')).toBe('opencode')
  })

  it('maps shared agent config to all', () => {
    expect(agentForPath('AGENTS.md')).toBe('all')
    expect(agentForPath('mcp.yaml')).toBe('all')
    expect(agentForPath('indexing.yaml')).toBe('all')
    expect(agentForPath('tools.yaml')).toBe('all')
    expect(agentForPath('skills-lock.json')).toBe('all')
    expect(agentForPath('config.d/foo.yaml')).toBe('all')
  })

  it('returns null for plain source files', () => {
    expect(agentForPath('src/main.go')).toBeNull()
    expect(agentForPath('cmd/skillgrid/main.go')).toBeNull()
    expect(agentForPath('README.md')).toBeNull()
  })
})
