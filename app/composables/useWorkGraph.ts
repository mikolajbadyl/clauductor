import type { NodeDetail, NodeStatus } from '~/types'


const layoutConfig = ref({
  mode: 'snake' as 'snake' | 'radial' | 'linear',
  snakeColumns: 5,
  radialSpacing: 200,
  nodeLimit: 50 as number | 'all'
})

// Module-scoped so all callers share the same state (e.g. ReadNode accessing results)
const nodeDetails = ref<Record<string, NodeDetail>>({})
const selectedNode = ref<NodeDetail | null>(null)

export function useWorkGraph() {
  const nodes = ref<any[]>([])
  const edges = ref<any[]>([])

  let idCounter = 0

  function nextId(prefix: string): string {
    return `${prefix}-${++idCounter}`
  }

  // Layout Constants
  const START_X = 100
  const START_Y = 100
  const Y_GAP = 180
  const SNAKE_GAP = 40
  const LINEAR_GAP = 60

  // Radial Constants
  const RADIAL_CENTER_X = 600
  const RADIAL_CENTER_Y = 400
  const BASE_RADIUS = 150

  // Estimated widths per node type to prevent overlaps
  function getNodeEstimatedWidth(node: any): number {
    return (node?.type === 'edit' || node?.type === 'read' || node?.type === 'write') ? 400 : 280
  }

  function computeSnakePositions(): { x: number, y: number }[] {
    const columns = layoutConfig.value.snakeColumns

    // Find widest node per visual column
    const columnWidths: number[] = Array(columns).fill(280)
    for (let i = 0; i < _nodes.length; i++) {
      const row = Math.floor(i / columns)
      const posInRow = i % columns
      const isEvenRow = row % 2 === 0
      const colIndex = isEvenRow ? posInRow : (columns - 1 - posInRow)
      columnWidths[colIndex] = Math.max(columnWidths[colIndex], getNodeEstimatedWidth(_nodes[i]))
    }

    // Accumulate column X positions based on actual widths
    const colX: number[] = []
    let x = START_X
    for (let c = 0; c < columns; c++) {
      colX.push(x)
      x += columnWidths[c] + SNAKE_GAP
    }

    return _nodes.map((_, index) => {
      const row = Math.floor(index / columns)
      const posInRow = index % columns
      const isEvenRow = row % 2 === 0
      const colIndex = isEvenRow ? posInRow : (columns - 1 - posInRow)
      return { x: colX[colIndex], y: START_Y + (row * Y_GAP) }
    })
  }

  function getConcentricPosition(index: number) {
    if (index === 0) return { x: RADIAL_CENTER_X, y: RADIAL_CENTER_Y }

    // Distance between complete loops (arms)
    const spacing = layoutConfig.value.radialSpacing || 100
    const b = spacing / (2 * Math.PI)
    // Distance between consecutive nodes along the spiral
    const D = 260

    // Start slightly outwards to avoid crushing early nodes in the center
    const theta_start = BASE_RADIUS / b

    // Archimedean Spiral constant arc-length approximation
    // Length = 1/2 * b * theta^2
    const theta = Math.sqrt(theta_start * theta_start + (2 * D * index) / b)
    const r = b * theta

    // Optional: add a tiny alternating offset to make the line zig-zag just a tiny bit so edges are distinct
    const bounceOffset = (index % 2 === 0 ? 8 : -8)

    return {
      x: RADIAL_CENTER_X + Math.cos(theta) * (r + bounceOffset),
      y: RADIAL_CENTER_Y + Math.sin(theta) * (r + bounceOffset)
    }
  }

  function computeLinearPositions(): { x: number, y: number }[] {
    const positions: { x: number, y: number }[] = []
    let x = START_X
    for (const node of _nodes) {
      positions.push({ x, y: RADIAL_CENTER_Y })
      x += getNodeEstimatedWidth(node) + LINEAR_GAP
    }
    return positions
  }

  function computeAllPositions(): { x: number, y: number }[] {
    switch (layoutConfig.value.mode) {
      case 'radial': return _nodes.map((_, i) => getConcentricPosition(i))
      case 'linear': return computeLinearPositions()
      case 'snake':
      default: return computeSnakePositions()
    }
  }

  function applyPositions() {
    const positions = computeAllPositions()
    _nodes = _nodes.map((node, i) => ({
      ...node,
      position: positions[i] || { x: START_X, y: START_Y }
    }))
  }

  let _nodes: any[] = []
  let _edges: any[] = []
  let _syncTimer: ReturnType<typeof requestAnimationFrame> | null = null
  const isLayouting = ref(false)

  function requestSync() {
    if (_syncTimer) return
    isLayouting.value = true
    _syncTimer = requestAnimationFrame(() => {
      let visibleNodes = [..._nodes]
      let visibleEdges = [..._edges]

      if (layoutConfig.value.nodeLimit !== 'all') {
        const limit = layoutConfig.value.nodeLimit as number
        if (_nodes.length > limit) {
          visibleNodes = _nodes.slice(-limit)
          const visibleIds = new Set(visibleNodes.map(n => n.id))
          visibleEdges = _edges.filter(e => visibleIds.has(e.source) && visibleIds.has(e.target))
        }
      }

      nodes.value = visibleNodes
      edges.value = visibleEdges
      _syncTimer = null

      setTimeout(() => {
        isLayouting.value = false
      }, 50)
    })
  }

  function addNode(id: string, label: string, nodeType: string, _x: number, _y: number, status: NodeStatus = 'idle', args?: Record<string, any>) {
    const nameLower = nodeType.toLowerCase()
    let flowNodeType: string
    let color: NodeDetail['color']
    if (nameLower.includes('edit') || nameLower.includes('replace')) {
      flowNodeType = 'edit'
      color = 'green'
    } else if (nameLower === 'write') {
      flowNodeType = 'write'
      color = 'green'
    } else if (nameLower === 'read') {
      flowNodeType = 'read'
      color = 'indigo'
    } else if (nameLower === 'websearch') {
      flowNodeType = 'websearch'
      color = 'amber'
    } else {
      flowNodeType = 'work'
      color = 'blue'
    }

    _nodes.push({
      id,
      type: flowNodeType,
      position: { x: 0, y: 0 },
      data: { label, icon: toolIcons[nodeType] || 'i-lucide-wrench', type: nodeType, status, args, color },
    })

    applyPositions()

    if (!nodeDetails.value[id]) {
      nodeDetails.value[id] = { id, type: nodeType, label, input: args, color }
    }
    requestSync()
  }

  function getSmartHandles(sourceId: string, targetId: string) {
    const sourceIndex = _nodes.findIndex(n => n.id === sourceId)
    const targetIndex = _nodes.findIndex(n => n.id === targetId)

    if (sourceIndex === -1 || targetIndex === -1) return { source: 'right-src', target: 'left' }

    if (layoutConfig.value.mode === 'snake') {
      const columns = layoutConfig.value.snakeColumns
      const sourceRow = Math.floor(sourceIndex / columns)
      const targetRow = Math.floor(targetIndex / columns)
      const isSourceEvenRow = sourceRow % 2 === 0

      if (sourceRow === targetRow) {
        return isSourceEvenRow ? { source: 'right-src', target: 'left' } : { source: 'left-src', target: 'right' }
      } else if (targetRow > sourceRow) {
        return { source: 'bottom', target: 'top' }
      }
    } else if (layoutConfig.value.mode === 'radial') {
      const sourceNode = _nodes[sourceIndex]
      const targetNode = _nodes[targetIndex]
      const dx = targetNode.position.x - sourceNode.position.x
      const dy = targetNode.position.y - sourceNode.position.y

      if (Math.abs(dx) > Math.abs(dy)) {
        return dx > 0 ? { source: 'right-src', target: 'left' } : { source: 'left', target: 'right-src' }
      } else {
        return dy > 0 ? { source: 'bottom', target: 'top' } : { source: 'top', target: 'bottom' }
      }
    }

    return { source: 'right-src', target: 'left' }
  }

  function addEdge(source: string, target: string, animated = false, sourceHandle?: string, targetHandle?: string) {
    const id = `e-${source}-${target}`
    if (_edges.find(e => e.id === id)) return

    let sH = sourceHandle
    let tH = targetHandle
    if (!sH || !tH) {
      const smart = getSmartHandles(source, target)
      sH = sH || smart.source
      tH = tH || smart.target
    }

    _edges.push({
      id,
      source,
      target,
      sourceHandle: sH,
      targetHandle: tH,
      type: 'bezier',
      animated,
      style: { stroke: '#1a2235', strokeWidth: 2 },
    })
    requestSync()
  }

  function updateNodeStatus(id: string, status: NodeStatus) {
    _nodes = _nodes.map(n =>
      n.id === id ? { ...n, data: { ...n.data, status } } : n
    )
    _edges = _edges.map(e => {
      if (e.target !== id) return e
      return {
        ...e,
        animated: status === 'running',
        style: {
          stroke: '#1a2235',
          strokeWidth: 2,
        },
      }
    })
    requestSync()
  }

  function updateNodeLabel(id: string, label: string) {
    _nodes = _nodes.map(n =>
      n.id === id ? { ...n, data: { ...n.data, label } } : n
    )
    if (nodeDetails.value[id]) {
      nodeDetails.value[id].label = label
    }
    requestSync()
  }

  function reset() {
    _nodes = []
    _edges = []
    nodes.value = []
    edges.value = []
    nodeDetails.value = {}
    selectedNode.value = null
    idCounter = 0
    if (_syncTimer) cancelAnimationFrame(_syncTimer)
    _syncTimer = null
  }

  function recalculatePositions() {
    applyPositions()

    _edges = _edges.map(edge => {
      const smart = getSmartHandles(edge.source, edge.target)
      return {
        ...edge,
        sourceHandle: smart.source,
        targetHandle: smart.target
      }
    })
    requestSync()
  }

  watch(layoutConfig, () => {
    recalculatePositions()
  }, { deep: true })

  return {
    nodes,
    edges,
    nodeDetails,
    selectedNode,
    nextId,
    addNode,
    addEdge,
    updateNodeStatus,
    updateNodeLabel,
    reset,
    layoutConfig,
    isLayouting,
  }
}
