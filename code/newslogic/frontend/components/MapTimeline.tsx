import { useState, useMemo, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { mapService, type MapEvent, type GeoJSONFeatureCollection, type GeoJSONFeature } from '@was/api-client'
import { format } from 'date-fns'
import MapLibreGL from 'maplibre-gl'
import { Map, MapControls, MapMarker, MarkerContent, useMap } from './ui/map'

interface MapTimelineProps {
  startDate?: Date
  endDate?: Date
  onEventClick?: (event: MapEvent) => void
}

/**
 * Map timeline component using mapcn.
 *
 * Displays geographic events on an interactive map with timeline filtering.
 * Uses the mapcn component library (MapLibre GL) for rendering.
 */
export function MapTimeline({ startDate, endDate, onEventClick }: MapTimelineProps) {
  const [selectedDate, setSelectedDate] = useState<Date | null>(null)
  const geojsonSourceId = 'geojson-events'
  const geojsonLayerId = 'geojson-events-layer'

  // Try to fetch GeoJSON first (preferred)
  const { data: geojsonData, isLoading: isLoadingGeoJSON } = useQuery({
    queryKey: ['map-geojson', startDate?.toISOString(), endDate?.toISOString()],
    queryFn: async () => {
      const params: Record<string, string | number> = {}
      if (startDate) params.start_date = startDate.toISOString()
      if (endDate) params.end_date = endDate.toISOString()
      return mapService.getGeoJSON(params)
    },
    retry: false, // Don't retry if GeoJSON fails, fallback to events
  })

  // Fallback: Fetch point-based events if GeoJSON is not available
  const { data: eventsData, isLoading: isLoadingEvents } = useQuery({
    queryKey: ['map-events', startDate?.toISOString(), endDate?.toISOString()],
    queryFn: async () => {
      const params: Record<string, string | number> = {}
      if (startDate) params.start_date = startDate.toISOString()
      if (endDate) params.end_date = endDate.toISOString()
      return mapService.getEvents(params)
    },
    enabled: !geojsonData, // Only fetch if GeoJSON is not available
  })

  const isLoading = isLoadingGeoJSON || (isLoadingEvents && !geojsonData)
  const events = eventsData?.events || []
  const geojson = geojsonData

  // Calculate map center and bounds from events or GeoJSON
  const mapCenter = useMemo(() => {
    if (geojson && geojson.features.length > 0) {
      // Calculate center from GeoJSON features
      let totalLat = 0
      let totalLon = 0
      let count = 0
      
      geojson.features.forEach((feature) => {
        const coords = feature.geometry.coordinates
        if (feature.geometry.type === 'Point') {
          const [lon, lat] = coords as number[]
          totalLon += lon
          totalLat += lat
          count++
        } else if (feature.geometry.type === 'LineString') {
          const lineCoords = coords as number[][]
          lineCoords.forEach(([lon, lat]) => {
            totalLon += lon
            totalLat += lat
            count++
          })
        } else if (feature.geometry.type === 'Polygon') {
          const polygonCoords = coords as number[][][]
          polygonCoords[0].forEach(([lon, lat]) => {
            totalLon += lon
            totalLat += lat
            count++
          })
        }
      })
      
      if (count > 0) {
        return [totalLon / count, totalLat / count] as [number, number]
      }
    }
    
    if (events.length === 0) return [0, 0] as [number, number]
    
    const avgLat = events.reduce((sum: number, e: MapEvent) => sum + e.latitude, 0) / events.length
    const avgLon = events.reduce((sum: number, e: MapEvent) => sum + e.longitude, 0) / events.length
    return [avgLon, avgLat] as [number, number]
  }, [events, geojson])

  // Filter events by selected date if timeline is being used
  const filteredEvents = useMemo(() => {
    if (!selectedDate) return events
    return events.filter((event: MapEvent) => {
      if (!event.occurred_at) return false
      const eventDate = new Date(event.occurred_at)
      return eventDate <= selectedDate
    })
  }, [events, selectedDate])

  // Filter GeoJSON features by selected date
  const filteredGeoJSON = useMemo(() => {
    if (!geojson) return null
    if (!selectedDate) return geojson
    
    const filteredFeatures = geojson.features.filter((feature) => {
      const occurredAt = feature.properties.occurred_at
      if (!occurredAt) return false
      const eventDate = new Date(occurredAt)
      return eventDate <= selectedDate
    })
    
    return {
      ...geojson,
      features: filteredFeatures,
    }
  }, [geojson, selectedDate])

  // Component to add GeoJSON source and layers to map
  function GeoJSONLayer({ geojson }: { geojson: GeoJSONFeatureCollection | null }) {
    const { map, isLoaded } = useMap()
    
    useEffect(() => {
      if (!map || !isLoaded || !geojson) return

      // Remove existing source and layer if they exist
      if (map.getSource(geojsonSourceId)) {
        if (map.getLayer(geojsonLayerId)) {
          map.removeLayer(geojsonLayerId)
        }
        map.removeSource(geojsonSourceId)
      }

      // Add GeoJSON source
      map.addSource(geojsonSourceId, {
        type: 'geojson',
        data: geojson as any, // Type assertion needed due to custom GeoJSON type
      })

      // Add layers for different geometry types
      // Points
      const pointFeatures = geojson.features.filter((f: GeoJSONFeature) => f.geometry.type === 'Point')
      if (pointFeatures.length > 0) {
        map.addLayer({
          id: `${geojsonLayerId}-points`,
          type: 'circle',
          source: geojsonSourceId,
          filter: ['==', ['geometry-type'], 'Point'],
          paint: {
            'circle-radius': 6,
            'circle-color': 'hsl(var(--primary))',
            'circle-stroke-width': 2,
            'circle-stroke-color': 'hsl(var(--background))',
          },
        })
      }

      // Lines
      const lineFeatures = geojson.features.filter((f: GeoJSONFeature) => f.geometry.type === 'LineString')
      if (lineFeatures.length > 0) {
        map.addLayer({
          id: `${geojsonLayerId}-lines`,
          type: 'line',
          source: geojsonSourceId,
          filter: ['==', ['geometry-type'], 'LineString'],
          paint: {
            'line-color': 'hsl(var(--primary))',
            'line-width': 2,
          },
        })
      }

      // Polygons
      const polygonFeatures = geojson.features.filter((f: GeoJSONFeature) => f.geometry.type === 'Polygon')
      if (polygonFeatures.length > 0) {
        map.addLayer({
          id: `${geojsonLayerId}-polygons`,
          type: 'fill',
          source: geojsonSourceId,
          filter: ['==', ['geometry-type'], 'Polygon'],
          paint: {
            'fill-color': 'hsl(var(--primary))',
            'fill-opacity': 0.3,
          },
        })
        map.addLayer({
          id: `${geojsonLayerId}-polygons-outline`,
          type: 'line',
          source: geojsonSourceId,
          filter: ['==', ['geometry-type'], 'Polygon'],
          paint: {
            'line-color': 'hsl(var(--primary))',
            'line-width': 2,
          },
        })
      }

      // Add click handler for all features
      const clickHandler = (e: MapLibreGL.MapLayerMouseEvent) => {
        if (e.features && e.features[0]) {
          const feature = e.features[0]
          const props = feature.properties as Record<string, unknown>
          if (onEventClick && props) {
            // Convert GeoJSON feature to MapEvent-like object
            const event: MapEvent = {
              id: (props.id as string) || '',
              title: (props.title as string) || '',
              location_name: (props.title as string) || '',
              latitude: 0, // Will be extracted from geometry if needed
              longitude: 0,
              occurred_at: (props.occurred_at as string) || null,
              entry_id: (props.entry_id as string) || '',
              entry_title: (props.entry_title as string) || '',
            }
            // Extract coordinates for point features
            if (feature.geometry.type === 'Point') {
              const [lon, lat] = feature.geometry.coordinates as number[]
              event.longitude = lon
              event.latitude = lat
            }
            onEventClick(event)
          }
        }
      }

      // Change cursor on hover
      const mouseEnterHandler = () => {
        map.getCanvas().style.cursor = 'pointer'
      }
      const mouseLeaveHandler = () => {
        map.getCanvas().style.cursor = ''
      }

      map.on('click', geojsonSourceId, clickHandler)
      map.on('mouseenter', geojsonSourceId, mouseEnterHandler)
      map.on('mouseleave', geojsonSourceId, mouseLeaveHandler)

      return () => {
        // Cleanup
        if (map.getLayer(`${geojsonLayerId}-points`)) {
          map.removeLayer(`${geojsonLayerId}-points`)
        }
        if (map.getLayer(`${geojsonLayerId}-lines`)) {
          map.removeLayer(`${geojsonLayerId}-lines`)
        }
        if (map.getLayer(`${geojsonLayerId}-polygons`)) {
          map.removeLayer(`${geojsonLayerId}-polygons`)
        }
        if (map.getLayer(`${geojsonLayerId}-polygons-outline`)) {
          map.removeLayer(`${geojsonLayerId}-polygons-outline`)
        }
        if (map.getSource(geojsonSourceId)) {
          map.removeSource(geojsonSourceId)
        }
        map.off('click', geojsonSourceId, clickHandler)
        map.off('mouseenter', geojsonSourceId, mouseEnterHandler)
        map.off('mouseleave', geojsonSourceId, mouseLeaveHandler)
      }
    }, [map, isLoaded, geojson, onEventClick])

    return null
  }

  return (
    <div className="flex h-full flex-col">
      {/* Timeline slider */}
      {(startDate || endDate) && (
        <div className="border-border bg-card border-b p-4">
          <div className="flex items-center gap-4">
            <label className="text-sm font-medium">Timeline</label>
            <input
              type="range"
              min={startDate?.getTime() || 0}
              max={endDate?.getTime() || Date.now()}
              value={selectedDate?.getTime() || endDate?.getTime() || Date.now()}
              onChange={(e) => setSelectedDate(new Date(Number(e.target.value)))}
              className="flex-1"
            />
            {selectedDate && (
              <span className="text-sm text-muted-foreground">{format(selectedDate, 'MMM d, yyyy')}</span>
            )}
          </div>
        </div>
      )}

      {/* Map container */}
      <div className="flex-1 relative">
        {isLoading ? (
          <div className="absolute inset-0 flex items-center justify-center bg-background/50 z-10">
            <div className="text-muted-foreground">Loading map events...</div>
          </div>
        ) : (
          <Map center={mapCenter} zoom={(geojson?.features.length || events.length) > 0 ? 4 : 2}>
            <MapControls />
            {/* Render GeoJSON if available */}
            {filteredGeoJSON && <GeoJSONLayer geojson={filteredGeoJSON} />}
            {/* Fallback to point markers if no GeoJSON */}
            {!geojson && filteredEvents.map((event) => (
              <MapMarker
                key={event.id}
                longitude={event.longitude}
                latitude={event.latitude}
                onClick={() => onEventClick?.(event)}
              >
                <MarkerContent>
                  <div className="bg-primary text-primary-foreground px-2 py-1 rounded text-xs cursor-pointer hover:bg-primary/90">
                    {event.title}
                  </div>
                </MarkerContent>
              </MapMarker>
            ))}
          </Map>
        )}
      </div>

      {/* Events list */}
      {((geojson && geojson.features.length > 0) || events.length > 0) && (
        <div className="border-border bg-card border-t p-4">
          <div className="text-sm font-medium mb-2">
            {geojson
              ? `${filteredGeoJSON?.features.length || 0} of ${geojson.features.length} events shown`
              : `${filteredEvents.length} of ${events.length} events shown`}
          </div>
          <div className="max-h-32 overflow-y-auto space-y-1">
            {geojson
              ? (filteredGeoJSON?.features || []).map((feature: GeoJSONFeature) => (
                  <button
                    key={feature.id || feature.properties.id}
                    onClick={() => {
                      if (onEventClick) {
                        const props = feature.properties
                        const event: MapEvent = {
                          id: props.id || '',
                          title: props.title || '',
                          location_name: props.title || '',
                          latitude: 0,
                          longitude: 0,
                          occurred_at: props.occurred_at || null,
                          entry_id: props.entry_id || '',
                          entry_title: props.entry_title || '',
                        }
                        if (feature.geometry.type === 'Point') {
                          const [lon, lat] = feature.geometry.coordinates as number[]
                          event.longitude = lon
                          event.latitude = lat
                        }
                        onEventClick(event)
                      }
                    }}
                    className="text-left text-xs hover:bg-accent p-2 rounded w-full"
                  >
                    <div className="font-medium">{feature.properties.title}</div>
                    <div className="text-muted-foreground">{feature.geometry.type}</div>
                    {feature.properties.occurred_at && (
                      <div className="text-muted-foreground/60 text-[10px] mt-0.5">
                        {format(new Date(feature.properties.occurred_at), 'MMM d, yyyy')}
                      </div>
                    )}
                  </button>
                ))
              : filteredEvents.map((event: MapEvent) => (
                  <button
                    key={event.id}
                    onClick={() => onEventClick?.(event)}
                    className="text-left text-xs hover:bg-accent p-2 rounded w-full"
                  >
                    <div className="font-medium">{event.title}</div>
                    <div className="text-muted-foreground">{event.location_name}</div>
                    {event.occurred_at && (
                      <div className="text-muted-foreground/60 text-[10px] mt-0.5">
                        {format(new Date(event.occurred_at), 'MMM d, yyyy')}
                      </div>
                    )}
                  </button>
                ))}
          </div>
        </div>
      )}
    </div>
  )
}

