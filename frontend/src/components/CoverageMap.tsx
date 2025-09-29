import { useEffect, useRef } from 'react';
import { MapContainer, TileLayer, Circle, Marker, Popup } from 'react-leaflet';
import { Icon } from 'leaflet';
import { MapPin } from 'lucide-react';
import 'leaflet/dist/leaflet.css';

// Fix for default markers in React Leaflet
import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';

delete (Icon.Default.prototype as any)._getIconUrl;
Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

interface CoverageMapProps {
  latitude: number | null;
  longitude: number | null;
  radiusKm: number;
  storeName?: string;
  className?: string;
}

export default function CoverageMap({ 
  latitude, 
  longitude, 
  radiusKm, 
  storeName = "Sua Loja",
  className = "" 
}: CoverageMapProps) {
  const mapRef = useRef<any>(null);

  // Return null if coordinates are not available
  if (!latitude || !longitude) {
    return (
      <div className={`h-64 w-full border rounded-lg flex items-center justify-center bg-muted/30 ${className}`}>
        <div className="text-center text-muted-foreground">
          <MapPin className="w-8 h-8 mx-auto mb-2 opacity-50" />
          <p className="text-sm">Coordenadas não configuradas</p>
          <p className="text-xs">Configure o endereço da loja para ver o mapa</p>
        </div>
      </div>
    );
  }

  useEffect(() => {
    // Fit bounds to include the circle when map loads
    if (mapRef.current) {
      const map = mapRef.current;
      if (radiusKm > 0) {
        // Calculate bounds to include the entire circle
        const bounds = [
          [latitude - (radiusKm / 111), longitude - (radiusKm / (111 * Math.cos(latitude * Math.PI / 180)))],
          [latitude + (radiusKm / 111), longitude + (radiusKm / (111 * Math.cos(latitude * Math.PI / 180)))]
        ];
        map.fitBounds(bounds, { padding: [20, 20] });
      } else {
        // If no radius, just center on the store
        map.setView([latitude, longitude], 13);
      }
    }
  }, [latitude, longitude, radiusKm]);

  // Convert km to meters for Leaflet Circle
  const radiusMeters = radiusKm * 1000;

  return (
    <div className={`h-64 w-full border rounded-lg overflow-hidden ${className}`} style={{ height: '580px' }}>
      <MapContainer
        center={[latitude, longitude]}
        zoom={13}
        style={{ height: '100%', width: '100%' }}
        ref={mapRef}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        
        {/* Store marker */}
        <Marker position={[latitude, longitude]}>
          <Popup>
            <div className="text-center">
              <strong>{storeName}</strong>
              <br />
              <small>
                Lat: {latitude.toFixed(6)}<br />
                Lng: {longitude.toFixed(6)}
              </small>
              {radiusKm > 0 && (
                <>
                  <br />
                  <small>Raio de cobertura: {radiusKm}km</small>
                </>
              )}
            </div>
          </Popup>
        </Marker>

        {/* Coverage circle - only show if radius > 0 */}
        {radiusKm > 0 && (
          <Circle
            center={[latitude, longitude]}
            radius={radiusMeters}
            pathOptions={{
              color: '#3b82f6',
              fillColor: '#3b82f6',
              fillOpacity: 0.1,
              weight: 2,
              dashArray: '5, 5'
            }}
          />
        )}
      </MapContainer>
    </div>
  );
}
