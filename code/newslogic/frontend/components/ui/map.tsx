import MapLibreGL, { type MarkerOptions } from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import {
  createContext,
  forwardRef,
  useCallback,
  useContext,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { Minus, Plus, Locate, Maximize, Loader2 } from 'lucide-react'

import { cn } from '@was/ui/utils'
import { useThemeStore } from '../../stores/themeStore'

type MapContextValue = {
  map: MapLibreGL.Map | null
  isLoaded: boolean
}

const MapContext = createContext<MapContextValue | null>(null)

function useMap() {
  const context = useContext(MapContext)
  if (!context) {
    throw new Error('useMap must be used within a Map component')
  }
  return context
}

const defaultStyles = {
  dark: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
  light: 'https://basemaps.cartocdn.com/gl/positron-gl-style/style.json',
}

type MapStyleOption = string | MapLibreGL.StyleSpecification

type MapProps = {
  children?: ReactNode
  /** Custom map styles for light and dark themes. Overrides the default Carto styles. */
  styles?: {
    light?: MapStyleOption
    dark?: MapStyleOption
  }
} & Omit<MapLibreGL.MapOptions, 'container' | 'style'>

type MapRef = MapLibreGL.Map

const DefaultLoader = () => (
  <div className="absolute inset-0 flex items-center justify-center">
    <div className="flex gap-1">
      <span className="size-1.5 rounded-full bg-muted-foreground/60 animate-pulse" />
      <span className="size-1.5 rounded-full bg-muted-foreground/60 animate-pulse [animation-delay:150ms]" />
      <span className="size-1.5 rounded-full bg-muted-foreground/60 animate-pulse [animation-delay:300ms]" />
    </div>
  </div>
)

const Map = forwardRef<MapRef, MapProps>(function Map({ children, styles, ...props }, ref) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [mapInstance, setMapInstance] = useState<MapLibreGL.Map | null>(null)
  const [isLoaded, setIsLoaded] = useState(false)
  const [isStyleLoaded, setIsStyleLoaded] = useState(false)
  const { resolvedTheme } = useThemeStore()
  const currentStyleRef = useRef<MapStyleOption | null>(null)

  const mapStyles = useMemo(
    () => ({
      dark: styles?.dark ?? defaultStyles.dark,
      light: styles?.light ?? defaultStyles.light,
    }),
    [styles]
  )

  useImperativeHandle(ref, () => mapInstance as MapLibreGL.Map, [mapInstance])

  useEffect(() => {
    if (!containerRef.current) return

    const initialStyle = resolvedTheme === 'dark' ? mapStyles.dark : mapStyles.light
    currentStyleRef.current = initialStyle

    const map = new MapLibreGL.Map({
      container: containerRef.current,
      style: initialStyle,
      renderWorldCopies: false,
      attributionControl: {
        compact: true,
      },
      ...props,
    })

    const styleDataHandler = () => setIsStyleLoaded(true)
    const loadHandler = () => setIsLoaded(true)

    map.on('load', loadHandler)
    map.on('styledata', styleDataHandler)
    setMapInstance(map)

    return () => {
      map.off('load', loadHandler)
      map.off('styledata', styleDataHandler)
      map.remove()
      setIsLoaded(false)
      setIsStyleLoaded(false)
      setMapInstance(null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!mapInstance || !resolvedTheme) return

    const newStyle = resolvedTheme === 'dark' ? mapStyles.dark : mapStyles.light

    if (currentStyleRef.current === newStyle) return

    currentStyleRef.current = newStyle
    setIsStyleLoaded(false)

    const frameId = requestAnimationFrame(() => {
      mapInstance.setStyle(newStyle, { diff: true })
    })

    return () => cancelAnimationFrame(frameId)
  }, [mapInstance, resolvedTheme, mapStyles])

  const isLoading = !isLoaded || !isStyleLoaded

  const contextValue = useMemo(
    () => ({
      map: mapInstance,
      isLoaded: isLoaded && isStyleLoaded,
    }),
    [mapInstance, isLoaded, isStyleLoaded]
  )

  return (
    <MapContext.Provider value={contextValue}>
      <div ref={containerRef} className="relative w-full h-full">
        {isLoading && <DefaultLoader />}
        {/* SSR-safe: children render only when map is loaded on client */}
        {mapInstance && children}
      </div>
    </MapContext.Provider>
  )
})

type MarkerContextValue = {
  marker: MapLibreGL.Marker
  map: MapLibreGL.Map | null
}

const MarkerContext = createContext<MarkerContextValue | null>(null)

function useMarkerContext() {
  const context = useContext(MarkerContext)
  if (!context) {
    throw new Error('Marker components must be used within MapMarker')
  }
  return context
}

type MapMarkerProps = {
  /** Longitude coordinate for marker position */
  longitude: number
  /** Latitude coordinate for marker position */
  latitude: number
  /** Marker subcomponents (MarkerContent, MarkerPopup, MarkerTooltip, MarkerLabel) */
  children: ReactNode
  /** Callback when marker is clicked */
  onClick?: (e: MouseEvent) => void
  /** Callback when mouse enters marker */
  onMouseEnter?: (e: MouseEvent) => void
  /** Callback when mouse leaves marker */
  onMouseLeave?: (e: MouseEvent) => void
  /** Callback when marker drag starts (requires draggable: true) */
  onDragStart?: (lngLat: { lng: number; lat: number }) => void
  /** Callback during marker drag (requires draggable: true) */
  onDrag?: (lngLat: { lng: number; lat: number }) => void
  /** Callback when marker drag ends (requires draggable: true) */
  onDragEnd?: (lngLat: { lng: number; lat: number }) => void
} & Omit<MarkerOptions, 'element'>

function MapMarker({
  longitude,
  latitude,
  children,
  onClick,
  onMouseEnter,
  onMouseLeave,
  onDragStart,
  onDrag,
  onDragEnd,
  draggable = false,
  ...markerOptions
}: MapMarkerProps) {
  const { map } = useMap()

  const marker = useMemo(() => {
    const markerInstance = new MapLibreGL.Marker({
      ...markerOptions,
      element: document.createElement('div'),
      draggable,
    }).setLngLat([longitude, latitude])

    const handleClick = (e: MouseEvent) => onClick?.(e)
    const handleMouseEnter = (e: MouseEvent) => onMouseEnter?.(e)
    const handleMouseLeave = (e: MouseEvent) => onMouseLeave?.(e)

    markerInstance.getElement()?.addEventListener('click', handleClick)
    markerInstance.getElement()?.addEventListener('mouseenter', handleMouseEnter)
    markerInstance.getElement()?.addEventListener('mouseleave', handleMouseLeave)

    const handleDragStart = () => {
      const lngLat = markerInstance.getLngLat()
      onDragStart?.({ lng: lngLat.lng, lat: lngLat.lat })
    }
    const handleDrag = () => {
      const lngLat = markerInstance.getLngLat()
      onDrag?.({ lng: lngLat.lng, lat: lngLat.lat })
    }
    const handleDragEnd = () => {
      const lngLat = markerInstance.getLngLat()
      onDragEnd?.({ lng: lngLat.lng, lat: lngLat.lat })
    }

    markerInstance.on('dragstart', handleDragStart)
    markerInstance.on('drag', handleDrag)
    markerInstance.on('dragend', handleDragEnd)

    return markerInstance

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!map) return

    marker.addTo(map)

    return () => {
      marker.remove()
    }

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [map])

  if (marker.getLngLat().lng !== longitude || marker.getLngLat().lat !== latitude) {
    marker.setLngLat([longitude, latitude])
  }
  if (marker.isDraggable() !== draggable) {
    marker.setDraggable(draggable)
  }

  const currentOffset = marker.getOffset()
  const newOffset = markerOptions.offset ?? [0, 0]
  const [newOffsetX, newOffsetY] = Array.isArray(newOffset)
    ? newOffset
    : [newOffset.x, newOffset.y]
  if (currentOffset.x !== newOffsetX || currentOffset.y !== newOffsetY) {
    marker.setOffset(newOffset)
  }

  if (marker.getRotation() !== markerOptions.rotation) {
    marker.setRotation(markerOptions.rotation ?? 0)
  }
  if (marker.getRotationAlignment() !== markerOptions.rotationAlignment) {
    marker.setRotationAlignment(markerOptions.rotationAlignment ?? 'auto')
  }
  if (marker.getPitchAlignment() !== markerOptions.pitchAlignment) {
    marker.setPitchAlignment(markerOptions.pitchAlignment ?? 'auto')
  }

  return <MarkerContext.Provider value={{ marker, map }}>{children}</MarkerContext.Provider>
}

type MarkerContentProps = {
  /** Custom marker content. Defaults to a blue dot if not provided */
  children?: ReactNode
  /** Additional CSS classes for the marker container */
  className?: string
}

function MarkerContent({ children, className }: MarkerContentProps) {
  const { marker } = useMarkerContext()

  return createPortal(
    <div className={cn('relative cursor-pointer', className)}>
      {children || <DefaultMarkerIcon />}
    </div>,
    marker.getElement()
  )
}

function DefaultMarkerIcon() {
  return (
    <div className="relative h-4 w-4 rounded-full border-2 border-white bg-blue-500 shadow-lg" />
  )
}

type MapControlsProps = {
  /** Position of the controls on the map (default: "bottom-right") */
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right'
  /** Show zoom in/out buttons (default: true) */
  showZoom?: boolean
  /** Show compass button to reset bearing (default: false) */
  showCompass?: boolean
  /** Show locate button to find user's location (default: false) */
  showLocate?: boolean
  /** Show fullscreen toggle button (default: false) */
  showFullscreen?: boolean
  /** Additional CSS classes for the controls container */
  className?: string
  /** Callback with user coordinates when located */
  onLocate?: (coords: { longitude: number; latitude: number }) => void
}

const positionClasses = {
  'top-left': 'top-2 left-2',
  'top-right': 'top-2 right-2',
  'bottom-left': 'bottom-2 left-2',
  'bottom-right': 'bottom-10 right-2',
}

function ControlGroup({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col rounded-md border border-border bg-background shadow-sm overflow-hidden [&>button:not(:last-child)]:border-b [&>button:not(:last-child)]:border-border">
      {children}
    </div>
  )
}

function ControlButton({
  onClick,
  label,
  children,
  disabled = false,
}: {
  onClick: () => void
  label: string
  children: React.ReactNode
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      aria-label={label}
      type="button"
      className={cn(
        'flex items-center justify-center size-8 hover:bg-accent dark:hover:bg-accent/40 transition-colors',
        disabled && 'opacity-50 pointer-events-none cursor-not-allowed'
      )}
      disabled={disabled}
    >
      {children}
    </button>
  )
}

function MapControls({
  position = 'bottom-right',
  showZoom = true,
  showCompass = false,
  showLocate = false,
  showFullscreen = false,
  className,
  onLocate,
}: MapControlsProps) {
  const { map, isLoaded } = useMap()
  const [waitingForLocation, setWaitingForLocation] = useState(false)

  const handleZoomIn = useCallback(() => {
    map?.zoomTo(map.getZoom() + 1, { duration: 300 })
  }, [map])

  const handleZoomOut = useCallback(() => {
    map?.zoomTo(map.getZoom() - 1, { duration: 300 })
  }, [map])

  const handleResetBearing = useCallback(() => {
    map?.resetNorthPitch({ duration: 300 })
  }, [map])

  const handleLocate = useCallback(() => {
    setWaitingForLocation(true)
    if ('geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          const coords = {
            longitude: pos.coords.longitude,
            latitude: pos.coords.latitude,
          }
          map?.flyTo({
            center: [coords.longitude, coords.latitude],
            zoom: 14,
            duration: 1500,
          })
          onLocate?.(coords)
          setWaitingForLocation(false)
        },
        (error) => {
          console.error('Error getting location:', error)
          setWaitingForLocation(false)
        }
      )
    }
  }, [map, onLocate])

  const handleFullscreen = useCallback(() => {
    const container = map?.getContainer()
    if (!container) return
    if (document.fullscreenElement) {
      document.exitFullscreen()
    } else {
      container.requestFullscreen()
    }
  }, [map])

  if (!isLoaded) return null

  return (
    <div
      className={cn('absolute z-10 flex flex-col gap-1.5', positionClasses[position], className)}
    >
      {showZoom && (
        <ControlGroup>
          <ControlButton onClick={handleZoomIn} label="Zoom in">
            <Plus className="size-4" />
          </ControlButton>
          <ControlButton onClick={handleZoomOut} label="Zoom out">
            <Minus className="size-4" />
          </ControlButton>
        </ControlGroup>
      )}
      {showCompass && (
        <ControlGroup>
          <CompassButton onClick={handleResetBearing} />
        </ControlGroup>
      )}
      {showLocate && (
        <ControlGroup>
          <ControlButton
            onClick={handleLocate}
            label="Find my location"
            disabled={waitingForLocation}
          >
            {waitingForLocation ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Locate className="size-4" />
            )}
          </ControlButton>
        </ControlGroup>
      )}
      {showFullscreen && (
        <ControlGroup>
          <ControlButton onClick={handleFullscreen} label="Toggle fullscreen">
            <Maximize className="size-4" />
          </ControlButton>
        </ControlGroup>
      )}
    </div>
  )
}

function CompassButton({ onClick }: { onClick: () => void }) {
  const { isLoaded, map } = useMap()
  const compassRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    if (!isLoaded || !map || !compassRef.current) return

    const compass = compassRef.current

    const updateRotation = () => {
      const bearing = map.getBearing()
      const pitch = map.getPitch()
      compass.style.transform = `rotateX(${pitch}deg) rotateZ(${-bearing}deg)`
    }

    map.on('rotate', updateRotation)
    map.on('pitch', updateRotation)
    updateRotation()

    return () => {
      map.off('rotate', updateRotation)
      map.off('pitch', updateRotation)
    }
  }, [isLoaded, map])

  return (
    <ControlButton onClick={onClick} label="Reset bearing to north">
      <svg
        ref={compassRef}
        viewBox="0 0 24 24"
        className="size-5 transition-transform duration-200"
        style={{ transformStyle: 'preserve-3d' }}
      >
        <path d="M12 2L16 12H12V2Z" className="fill-red-500" />
        <path d="M12 2L8 12H12V2Z" className="fill-red-300" />
        <path d="M12 22L16 12H12V22Z" className="fill-muted-foreground/60" />
        <path d="M12 22L8 12H12V22Z" className="fill-muted-foreground/30" />
      </svg>
    </ControlButton>
  )
}

export { Map, useMap, MapMarker, MarkerContent, MapControls }

