import type { RoadmapNode, RoadmapTreeNode } from '@/types'

/**
 * Tidy-tree layout for the roadmap canvas: roots at the left, branches flow
 * right. Depth fixes x; a post-order walk assigns y by leaf slots so siblings
 * never overlap and parents center over their children. Pure — no DOM, no Vue —
 * so it is unit-testable and SSR-safe.
 */

export const CARD_W = 244
export const CARD_H = 96
export const H_GAP = 72
export const V_GAP = 18
/** Extra vertical room between separate roots so trunks read as distinct. */
export const ROOT_GAP = 28

export interface LaidOutNode {
  node: RoadmapTreeNode
  /** Top-left corner of the card in world coordinates. */
  x: number
  y: number
  depth: number
}

export interface LayoutEdge {
  fromId: string
  toId: string
}

export interface TreeLayout {
  placed: LaidOutNode[]
  edges: LayoutEdge[]
  width: number
  height: number
}

/**
 * Assemble the flat API list into a sorted forest. Orphaned parents promote a
 * node to root; a final visited-set sweep rescues nodes trapped in a cycle
 * (corrupt data) so nothing silently vanishes from the canvas.
 */
export function buildForest(nodes: RoadmapNode[]): RoadmapTreeNode[] {
  const map = new Map<string, RoadmapTreeNode>()
  for (const n of nodes) map.set(n.id, { ...n, children: [] })
  const roots: RoadmapTreeNode[] = []
  for (const n of map.values()) {
    if (n.parentId && map.has(n.parentId) && n.parentId !== n.id) {
      map.get(n.parentId)!.children.push(n)
    } else {
      roots.push(n)
    }
  }
  // Cycle rescue: mark everything reachable from roots; any node left over is
  // stuck in a loop — cut it from its parent's children and re-seat it as a
  // root, so rendering terminates and the node stays visible. (Single-parent
  // links mean a cycle is always unreachable, so one cut per leftover node
  // breaks every loop.)
  const reachable = new Set<string>()
  const mark = (n: RoadmapTreeNode) => {
    if (reachable.has(n.id)) return
    reachable.add(n.id)
    for (const c of n.children) mark(c)
  }
  for (const r of roots) mark(r)
  for (const n of map.values()) {
    if (reachable.has(n.id)) continue
    const parent = n.parentId ? map.get(n.parentId) : undefined
    if (parent) parent.children = parent.children.filter((c) => c.id !== n.id)
    roots.push(n)
    mark(n)
  }
  const sortRec = (list: RoadmapTreeNode[]) => {
    list.sort((a, b) => a.position - b.position)
    for (const x of list) sortRec(x.children)
  }
  sortRec(roots)
  return roots
}

/** True when `maybeDescendant` sits under `ancestorId` in the parent chain. */
export function isDescendant(nodes: RoadmapNode[], ancestorId: string, maybeDescendant: string): boolean {
  const parentOf = new Map(nodes.map((n) => [n.id, n.parentId]))
  let cur = parentOf.get(maybeDescendant) ?? null
  let guard = 0
  while (cur && guard++ <= nodes.length) {
    if (cur === ancestorId) return true
    cur = parentOf.get(cur) ?? null
  }
  return false
}

/** Lay out the whole forest. Empty input yields a zero-size layout. */
export function layoutForest(nodes: RoadmapNode[]): TreeLayout {
  const roots = buildForest(nodes)
  const placed: LaidOutNode[] = []
  const edges: LayoutEdge[] = []
  let cursor = 0 // y of the next free leaf slot
  let maxDepth = 0

  // Post-order: children first, parent centered over its first..last child.
  // Returns the y assigned to this node.
  const place = (node: RoadmapTreeNode, depth: number): number => {
    maxDepth = Math.max(maxDepth, depth)
    const x = depth * (CARD_W + H_GAP)
    let y: number
    if (node.children.length === 0) {
      y = cursor
      cursor += CARD_H + V_GAP
    } else {
      const childYs = node.children.map((child) => {
        edges.push({ fromId: node.id, toId: child.id })
        return place(child, depth + 1)
      })
      y = (childYs[0]! + childYs[childYs.length - 1]!) / 2
    }
    placed.push({ node, x, y, depth })
    return y
  }

  roots.forEach((root, i) => {
    if (i > 0) cursor += ROOT_GAP
    place(root, 0)
  })

  return {
    placed,
    edges,
    width: placed.length ? (maxDepth + 1) * (CARD_W + H_GAP) - H_GAP : 0,
    height: placed.length ? Math.max(...placed.map((p) => p.y)) + CARD_H : 0,
  }
}

export interface SiblingBand {
  /** The parent whose children this band groups. */
  parentId: string
  x: number
  y: number
  w: number
  h: number
}

/** Quiet padding around a sibling set so the panel breathes. */
export const BAND_PAD = 14

/**
 * A faint panel behind each set of ≥2 sibling cards ("parallel tracks"),
 * anchoring branches against the blank 留白 ground (collab Workstream D).
 * Roots are deliberately NOT banded — ROOT_GAP already separates trunks, and a
 * single band spanning every root would just repaint the canvas. Siblings share
 * a depth (hence one column x), so the band is a single rounded rect over their
 * contiguous y-range. Pure + SSR-safe.
 */
export function siblingBands(placed: LaidOutNode[]): SiblingBand[] {
  const byParent = new Map<string, LaidOutNode[]>()
  for (const p of placed) {
    const key = p.node.parentId ?? ''
    if (key === '') continue
    const list = byParent.get(key)
    if (list) list.push(p)
    else byParent.set(key, [p])
  }
  const bands: SiblingBand[] = []
  for (const [parentId, group] of byParent) {
    if (group.length < 2) continue
    let minY = Infinity
    let maxY = -Infinity
    for (const p of group) {
      if (p.y < minY) minY = p.y
      if (p.y > maxY) maxY = p.y
    }
    bands.push({
      parentId,
      x: group[0]!.x - BAND_PAD,
      y: minY - BAND_PAD,
      w: CARD_W + BAND_PAD * 2,
      h: maxY - minY + CARD_H + BAND_PAD * 2,
    })
  }
  return bands
}
