import { describe, expect, it } from 'vitest'
import type { RoadmapNode, RoadmapStatus, RoadmapTreeNode } from '@/types'
import { buildForest, isDescendant, layoutForest, siblingBands, BAND_PAD, CARD_H, CARD_W, H_GAP, V_GAP, ROOT_GAP } from '@/utils/tidyTree'

// tidyTree is pure — no backend, no mocks; these run anywhere.

let seq = 0
function n(over: Partial<RoadmapNode> & { id: string }): RoadmapNode {
  return {
    projectId: 'p1',
    parentId: null,
    title: over.id,
    description: '',
    status: 'pending' as RoadmapStatus,
    position: seq++,
    createdAt: '2026-07-24T00:00:00Z',
    updatedAt: '2026-07-24T00:00:00Z',
    ...over,
  }
}
const at = (layout: ReturnType<typeof layoutForest>, id: string) => layout.placed.find((p) => p.node.id === id)!

describe('buildForest', () => {
  it('assembles children and sorts by position', () => {
    const roots = buildForest([
      n({ id: 'b', position: 2 }),
      n({ id: 'a', position: 1 }),
      n({ id: 'a1', parentId: 'a', position: 0 }),
    ])
    expect(roots.map((r) => r.id)).toEqual(['a', 'b'])
    expect(roots[0]!.children.map((c) => c.id)).toEqual(['a1'])
  })

  it('promotes orphans (missing parent) to roots', () => {
    const roots = buildForest([n({ id: 'x', parentId: 'ghost' })])
    expect(roots.map((r) => r.id)).toEqual(['x'])
  })

  it('rescues nodes trapped in a cycle instead of dropping them', () => {
    // A 2-cycle has no valid root; the rescue cuts one edge and keeps BOTH
    // nodes visible as a single-rooted chain (and rendering terminates).
    const roots = buildForest([n({ id: 'a', parentId: 'b' }), n({ id: 'b', parentId: 'a' })])
    expect(roots).toHaveLength(1)
    const all: string[] = []
    const walk = (x: RoadmapTreeNode) => {
      all.push(x.id)
      x.children.forEach(walk)
    }
    roots.forEach(walk)
    expect(all.sort()).toEqual(['a', 'b'])
  })
})

describe('isDescendant', () => {
  const chain = [
    n({ id: 'root' }),
    n({ id: 'mid', parentId: 'root' }),
    n({ id: 'leaf', parentId: 'mid' }),
    n({ id: 'other' }),
  ]
  it('finds transitive descendants', () => {
    expect(isDescendant(chain, 'root', 'leaf')).toBe(true)
    expect(isDescendant(chain, 'mid', 'leaf')).toBe(true)
  })
  it('rejects non-descendants and self', () => {
    expect(isDescendant(chain, 'leaf', 'root')).toBe(false)
    expect(isDescendant(chain, 'root', 'root')).toBe(false)
    expect(isDescendant(chain, 'root', 'other')).toBe(false)
  })
})

describe('layoutForest', () => {
  it('handles the empty forest', () => {
    const layout = layoutForest([])
    expect(layout.placed).toEqual([])
    expect(layout.edges).toEqual([])
    expect(layout.width).toBe(0)
    expect(layout.height).toBe(0)
  })

  it('places a single node at the origin', () => {
    const layout = layoutForest([n({ id: 'solo' })])
    expect(at(layout, 'solo')).toMatchObject({ x: 0, y: 0, depth: 0 })
    expect(layout.width).toBe(CARD_W)
    expect(layout.height).toBe(CARD_H)
  })

  it('lays children one depth-step right and centers the parent over them', () => {
    seq = 0
    const layout = layoutForest([
      n({ id: 'root' }),
      n({ id: 'c1', parentId: 'root' }),
      n({ id: 'c2', parentId: 'root' }),
    ])
    expect(at(layout, 'c1').x).toBe(CARD_W + H_GAP)
    expect(at(layout, 'c1').y).toBe(0)
    expect(at(layout, 'c2').y).toBe(CARD_H + V_GAP)
    expect(at(layout, 'root').y).toBe((CARD_H + V_GAP) / 2)
    expect(layout.edges).toHaveLength(2)
    expect(layout.width).toBe(2 * CARD_W + H_GAP)
  })

  it('keeps extra room between separate roots', () => {
    seq = 0
    const layout = layoutForest([n({ id: 'r1' }), n({ id: 'r2' })])
    expect(at(layout, 'r2').y).toBe(CARD_H + V_GAP + ROOT_GAP)
  })

  it('never overlaps siblings in deep chains', () => {
    seq = 0
    const nodes = [
      n({ id: 'r' }),
      n({ id: 'a', parentId: 'r' }),
      n({ id: 'b', parentId: 'r' }),
      n({ id: 'a1', parentId: 'a' }),
      n({ id: 'a2', parentId: 'a' }),
      n({ id: 'b1', parentId: 'b' }),
    ]
    const layout = layoutForest(nodes)
    const ys = layout.placed.map((p) => p.y)
    // Leaf slots are distinct: no two cards share a top edge at the same depth.
    const depthOf = (id: string) => at(layout, id).depth
    for (let i = 0; i < layout.placed.length; i++) {
      for (let j = i + 1; j < layout.placed.length; j++) {
        const p = layout.placed[i]!
        const q = layout.placed[j]!
        if (depthOf(p.node.id) !== depthOf(q.node.id)) continue
        expect(Math.abs(p.y - q.y)).toBeGreaterThanOrEqual(CARD_H + V_GAP - 0.001)
      }
    }
    expect(ys.length).toBe(6)
    expect(at(layout, 'a').y).toBe((at(layout, 'a1').y + at(layout, 'a2').y) / 2)
  })
})

describe('siblingBands', () => {
  it('returns nothing for an empty forest', () => {
    expect(siblingBands(layoutForest([]).placed)).toEqual([])
  })

  it('skips lone children and never bands roots', () => {
    seq = 0
    const layout = layoutForest([
      n({ id: 'r1' }),
      n({ id: 'r2' }),
      n({ id: 'only', parentId: 'r1' }),
    ])
    // r1 has one child (no band); r1/r2 are roots (separated by ROOT_GAP, not banded).
    expect(siblingBands(layout.placed)).toEqual([])
  })

  it('draws one tight panel per parent with ≥2 children', () => {
    seq = 0
    const layout = layoutForest([
      n({ id: 'root' }),
      n({ id: 'c1', parentId: 'root' }),
      n({ id: 'c2', parentId: 'root' }),
      n({ id: 'c3', parentId: 'root' }),
    ])
    const bands = siblingBands(layout.placed)
    expect(bands).toHaveLength(1)
    const b = bands[0]!
    const c1 = at(layout, 'c1')
    const c3 = at(layout, 'c3')
    expect(b.parentId).toBe('root')
    expect(b.x).toBe(c1.x - BAND_PAD)
    expect(b.w).toBe(CARD_W + BAND_PAD * 2)
    expect(b.y).toBe(c1.y - BAND_PAD)
    expect(b.h).toBe(c3.y - c1.y + CARD_H + BAND_PAD * 2)
  })

  it('bands each qualifying parent separately', () => {
    seq = 0
    const layout = layoutForest([
      n({ id: 'root' }),
      n({ id: 'a', parentId: 'root' }),
      n({ id: 'b', parentId: 'root' }),
      n({ id: 'a1', parentId: 'a' }),
      n({ id: 'a2', parentId: 'a' }),
    ])
    // root→{a,b} and a→{a1,a2}: two sibling sets of size 2 → two bands.
    const parents = siblingBands(layout.placed).map((b) => b.parentId).sort()
    expect(parents).toEqual(['a', 'root'])
  })
})
