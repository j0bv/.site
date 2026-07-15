/**
 * Knowledge Graph Visualizer Component.
 *
 * Integrates Supermemory memory-graph package for visualizing
 * knowledge graph data from Neo4j.
 */

import { useQuery } from '@tanstack/react-query'
import { Network, X } from 'lucide-react'
import { memoryService, type MemoryGraphResponse } from '@was/api-client'
import { Button } from '@was/ui'

interface KnowledgeGraphProps {
  entryIds?: string[]
  onNodeClick?: (nodeId: string, nodeType: string) => void
  onClose?: () => void
}

/**
 * Knowledge Graph Visualizer.
 *
 * Displays an interactive graph visualization of entities and relationships.
 * Uses Supermemory memory-graph package when available, with fallback UI.
 */
export function KnowledgeGraph({ entryIds, onClose }: KnowledgeGraphProps) {

  // Fetch graph data
  const { data: graphData, isLoading, error } = useQuery<MemoryGraphResponse>({
    queryKey: ['memory-graph', entryIds?.join(',')],
    queryFn: async () => {
      const params: Record<string, string | number> = {}
      if (entryIds && entryIds.length > 0) {
        params.entry_ids = entryIds.join(',')
      }
      params.limit = 1000
      return memoryService.getMemoryGraph(params)
    },
    enabled: true,
  })

  // Try to use Supermemory memory-graph package if available
  // For now, we'll create a basic visualization
  // TODO: Integrate @supermemoryai/memory-graph when package is available

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <Network className="mx-auto h-12 w-12 animate-pulse text-muted-foreground" />
          <p className="mt-4 text-sm text-muted-foreground">Loading graph data...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <Network className="mx-auto h-12 w-12 text-destructive" />
          <p className="mt-4 text-sm text-destructive">Failed to load graph data</p>
          <p className="mt-2 text-xs text-muted-foreground">{String(error)}</p>
        </div>
      </div>
    )
  }

  if (!graphData || graphData.nodes.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <Network className="mx-auto h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-sm text-muted-foreground">No graph data available</p>
          <p className="mt-2 text-xs text-muted-foreground">
            Process some entries with NLP to see the knowledge graph
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-border bg-card flex items-center justify-between border-b p-4">
        <div>
          <h2 className="text-lg font-semibold">Knowledge Graph</h2>
          <p className="text-xs text-muted-foreground">
            {graphData.nodes.length} nodes, {graphData.edges.length} relationships
          </p>
        </div>
        {onClose && (
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>

      {/* Graph Visualization */}
      <div className="relative flex-1 overflow-hidden">
        {/* Placeholder for memory-graph component */}
        {/* TODO: Replace with actual Supermemory memory-graph component */}
        <div className="flex h-full items-center justify-center bg-muted/30">
          <div className="text-center">
            <Network className="mx-auto h-16 w-16 text-muted-foreground" />
            <p className="mt-4 text-sm font-medium">Graph Visualization</p>
            <p className="mt-2 text-xs text-muted-foreground">
              Install @supermemoryai/memory-graph to enable interactive visualization
            </p>
            <div className="mt-4 space-y-2 text-left text-xs">
              <div>
                <span className="font-medium">Nodes:</span> {graphData.nodes.length}
              </div>
              <div>
                <span className="font-medium">Edges:</span> {graphData.edges.length}
              </div>
              <div className="mt-4 max-h-40 overflow-y-auto">
                <div className="font-medium mb-2">Node Types:</div>
                {Array.from(new Set(graphData.nodes.map((n) => n.type))).map((type) => (
                  <div key={type} className="text-muted-foreground">
                    • {type} ({graphData.nodes.filter((n) => n.type === type).length})
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

