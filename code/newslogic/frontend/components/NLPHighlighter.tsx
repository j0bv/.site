import { useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { nlpService } from '@was/api-client'

interface NLPHighlighterProps {
  entryId: string
  content: string
  highlightEntities?: boolean
  highlightPOSTags?: boolean
}

/**
 * Component to highlight NLP results in article content.
 *
 * Highlights named entities and optionally POS tags in the rendered content.
 */
export function NLPHighlighter({ entryId, content, highlightEntities = true, highlightPOSTags = false }: NLPHighlighterProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  // Fetch NLP results
  const { data: nlpResult } = useQuery({
    queryKey: ['nlp-result', entryId],
    queryFn: () => nlpService.getEntryNLP(entryId),
    enabled: !!entryId,
  })

  useEffect(() => {
    if (!containerRef.current || !nlpResult) return

    const container = containerRef.current

    // Remove existing highlights
    container.querySelectorAll('.nlp-entity-highlight, .nlp-pos-highlight').forEach((el) => {
      const parent = el.parentNode
      if (parent) {
        parent.replaceChild(document.createTextNode(el.textContent || ''), el)
        parent.normalize()
      }
    })

    if (highlightEntities && nlpResult.entities) {
      // Sort entities by start position (descending) to avoid offset issues
      const sortedEntities = [...nlpResult.entities].sort((a, b) => b.start - a.start)

      sortedEntities.forEach((entity) => {
        // Find text node containing this entity
        const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null)
        let node: Text | null
        let offset = 0

        while ((node = walker.nextNode() as Text | null)) {
          const nodeLength = node.textContent?.length || 0
          const nodeStart = offset
          const nodeEnd = offset + nodeLength

          // Check if entity is within this text node
          if (entity.start >= nodeStart && entity.end <= nodeEnd) {
            const relativeStart = entity.start - nodeStart
            const relativeEnd = entity.end - nodeStart
            const text = node.textContent || ''

            // Split text node
            const before = text.substring(0, relativeStart)
            const entityText = text.substring(relativeStart, relativeEnd)
            const after = text.substring(relativeEnd)

            // Create highlight span
            const span = document.createElement('span')
            span.className = `nlp-entity-highlight nlp-entity-${entity.label.toLowerCase()}`
            span.textContent = entityText
            span.title = `${entity.label}: ${entityText}`

            // Replace text node with fragments
            const fragment = document.createDocumentFragment()
            if (before) fragment.appendChild(document.createTextNode(before))
            fragment.appendChild(span)
            if (after) fragment.appendChild(document.createTextNode(after))

            node.parentNode?.replaceChild(fragment, node)
            break
          }

          offset = nodeEnd
        }
      })
    }

    // Add CSS for entity highlighting
    if (!document.getElementById('nlp-highlight-styles')) {
      const style = document.createElement('style')
      style.id = 'nlp-highlight-styles'
      style.textContent = `
        .nlp-entity-highlight {
          padding: 2px 4px;
          border-radius: 3px;
          cursor: help;
        }
        .nlp-entity-person { background-color: rgba(59, 130, 246, 0.2); }
        .nlp-entity-org { background-color: rgba(16, 185, 129, 0.2); }
        .nlp-entity-gpe { background-color: rgba(245, 158, 11, 0.2); }
        .nlp-entity-loc { background-color: rgba(236, 72, 153, 0.2); }
        .nlp-entity-event { background-color: rgba(139, 92, 246, 0.2); }
      `
      document.head.appendChild(style)
    }
  }, [nlpResult, highlightEntities, highlightPOSTags, content])

  return <div ref={containerRef} dangerouslySetInnerHTML={{ __html: content }} />
}

