package services

import (
	"context"
	"encoding/json"
	"fmt"
	"iafarma/pkg/models"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// DeliveryService manages delivery zone validation and distance calculations
type DeliveryService struct {
	db               *gorm.DB
	googleMapsAPIKey string
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(db *gorm.DB, googleMapsAPIKey string) *DeliveryService {
	return &DeliveryService{
		db:               db,
		googleMapsAPIKey: googleMapsAPIKey,
	}
}

// GoogleMapsDistanceResponse represents the response from Google Maps Distance Matrix API
type GoogleMapsDistanceResponse struct {
	DestinationAddresses []string `json:"destination_addresses"`
	OriginAddresses      []string `json:"origin_addresses"`
	Rows                 []struct {
		Elements []struct {
			Distance struct {
				Text  string `json:"text"`
				Value int    `json:"value"` // distance in meters
			} `json:"distance"`
			Duration struct {
				Text  string `json:"text"`
				Value int    `json:"value"` // duration in seconds
			} `json:"duration"`
			Status string `json:"status"`
		} `json:"elements"`
	} `json:"rows"`
	Status string `json:"status"`
}

// GoogleMapsGeocodingResponse represents the response from Google Maps Geocoding API
type GoogleMapsGeocodingResponse struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}

// DeliveryValidationResult represents the result of delivery validation
type DeliveryValidationResult struct {
	CanDeliver      bool    `json:"can_deliver"`
	DistanceKm      float64 `json:"distance_km,omitempty"`
	DurationMinutes int     `json:"duration_minutes,omitempty"`
	Reason          string  `json:"reason,omitempty"`
	ZoneType        string  `json:"zone_type,omitempty"` // whitelist, blacklist, or normal
}

// ValidateDeliveryAddress validates if delivery can be made to the given address
func (s *DeliveryService) ValidateDeliveryAddress(ctx context.Context, tenantID uuid.UUID, customerAddress models.Address) (*DeliveryValidationResult, error) {
	// Get tenant configuration
	var tenant models.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// If store address is not configured, allow delivery everywhere
	if tenant.StoreLatitude == nil || tenant.StoreLongitude == nil {
		return &DeliveryValidationResult{
			CanDeliver: true,
			Reason:     "store_location_not_configured",
		}, nil
	}

	// Check neighborhood in whitelist/blacklist
	zoneResult := s.checkDeliveryZones(tenantID, customerAddress)
	if zoneResult != nil {
		return zoneResult, nil
	}

	// If no radius limit (0), allow delivery everywhere
	if tenant.DeliveryRadiusKm == 0 {
		return &DeliveryValidationResult{
			CanDeliver: true,
			Reason:     "no_radius_limit",
			ZoneType:   "normal",
		}, nil
	}

	// Calculate distance using Google Maps API
	customerAddressStr := s.formatAddressForGeocoding(customerAddress)
	storeAddressStr := fmt.Sprintf("%f,%f", *tenant.StoreLatitude, *tenant.StoreLongitude)

	distance, duration, err := s.calculateDistance(ctx, storeAddressStr, customerAddressStr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to calculate distance")
		// In case of API error, allow delivery (fallback)
		return &DeliveryValidationResult{
			CanDeliver: true,
			Reason:     "distance_calculation_failed",
			ZoneType:   "normal",
		}, nil
	}

	distanceKm := float64(distance) / 1000.0
	durationMin := duration / 60

	canDeliver := distanceKm <= float64(tenant.DeliveryRadiusKm)

	result := &DeliveryValidationResult{
		CanDeliver:      canDeliver,
		DistanceKm:      distanceKm,
		DurationMinutes: durationMin,
		ZoneType:        "normal",
	}

	if !canDeliver {
		result.Reason = "outside_delivery_radius"
	}

	return result, nil
}

// checkDeliveryZones checks if the address is in whitelist or blacklist
func (s *DeliveryService) checkDeliveryZones(tenantID uuid.UUID, address models.Address) *DeliveryValidationResult {
	var zones []models.TenantDeliveryZone

	// Check for exact neighborhood match
	query := s.db.Where("tenant_id = ?", tenantID)

	// Try exact neighborhood match first
	if address.Neighborhood != "" {
		err := query.Where("LOWER(neighborhood_name) = LOWER(?)", address.Neighborhood).Find(&zones).Error
		if err == nil && len(zones) > 0 {
			for _, zone := range zones {
				if zone.ZoneType == "blacklist" {
					return &DeliveryValidationResult{
						CanDeliver: false,
						Reason:     "area_not_served",
						ZoneType:   "blacklist",
					}
				} else if zone.ZoneType == "whitelist" {
					return &DeliveryValidationResult{
						CanDeliver: true,
						Reason:     "whitelisted_area",
						ZoneType:   "whitelist",
					}
				}
			}
		}
	}

	// If no specific zone found, continue with normal validation
	return nil
}

// calculateDistance calculates distance using Google Maps Distance Matrix API
func (s *DeliveryService) calculateDistance(ctx context.Context, origin, destination string) (distanceMeters int, durationSeconds int, err error) {
	if s.googleMapsAPIKey == "" {
		return 0, 0, fmt.Errorf("Google Maps API key not configured")
	}

	baseURL := "https://maps.googleapis.com/maps/api/distancematrix/json"
	params := url.Values{}
	params.Set("origins", origin)
	params.Set("destinations", destination)
	params.Set("units", "metric")
	params.Set("key", s.googleMapsAPIKey)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to call Google Maps API: %w", err)
	}
	defer resp.Body.Close()

	var result GoogleMapsDistanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("failed to decode Google Maps response: %w", err)
	}

	if result.Status != "OK" {
		return 0, 0, fmt.Errorf("Google Maps API error: %s", result.Status)
	}

	if len(result.Rows) == 0 || len(result.Rows[0].Elements) == 0 {
		return 0, 0, fmt.Errorf("no route found")
	}

	element := result.Rows[0].Elements[0]
	if element.Status != "OK" {
		return 0, 0, fmt.Errorf("route calculation failed: %s", element.Status)
	}

	return element.Distance.Value, element.Duration.Value, nil
}

// GeocodeAddress converts an address to coordinates using Google Maps Geocoding API
func (s *DeliveryService) GeocodeAddress(ctx context.Context, address string) (lat, lng float64, err error) {
	if s.googleMapsAPIKey == "" {
		return 0, 0, fmt.Errorf("Google Maps API key not configured")
	}

	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Set("address", address)
	params.Set("key", s.googleMapsAPIKey)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to call Google Maps Geocoding API: %w", err)
	}
	defer resp.Body.Close()

	var result GoogleMapsGeocodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("failed to decode Google Maps response: %w", err)
	}

	if result.Status != "OK" {
		return 0, 0, fmt.Errorf("Google Maps Geocoding API error: %s", result.Status)
	}

	if len(result.Results) == 0 {
		return 0, 0, fmt.Errorf("no coordinates found for address")
	}

	location := result.Results[0].Geometry.Location
	return location.Lat, location.Lng, nil
}

// formatAddressForGeocoding formats a models.Address for geocoding
func (s *DeliveryService) formatAddressForGeocoding(address models.Address) string {
	var parts []string

	if address.Street != "" {
		streetPart := address.Street
		if address.Number != "" {
			streetPart += ", " + address.Number
		}
		parts = append(parts, streetPart)
	}

	if address.Neighborhood != "" {
		parts = append(parts, address.Neighborhood)
	}

	if address.City != "" {
		parts = append(parts, address.City)
	}

	if address.State != "" {
		parts = append(parts, address.State)
	}

	if address.ZipCode != "" {
		parts = append(parts, address.ZipCode)
	}

	if address.Country != "" {
		parts = append(parts, address.Country)
	}

	return strings.Join(parts, ", ")
}

// UpdateTenantStoreLocation updates the store location and geocodes it
func (s *DeliveryService) UpdateTenantStoreLocation(ctx context.Context, tenantID uuid.UUID, storeAddress string, addressData map[string]interface{}) error {
	lat, lng, err := s.GeocodeAddress(ctx, storeAddress)
	if err != nil {
		return fmt.Errorf("failed to geocode store address: %w", err)
	}

	// Add coordinates to the update data
	addressData["store_latitude"] = lat
	addressData["store_longitude"] = lng

	return s.db.Model(&models.Tenant{}).Where("id = ?", tenantID).Updates(addressData).Error
}

// ManageDeliveryZone adds or removes a neighborhood from whitelist/blacklist
func (s *DeliveryService) ManageDeliveryZone(tenantID uuid.UUID, neighborhood, city, state, zoneType string, add bool) error {
	if add {
		zone := models.TenantDeliveryZone{
			BaseModel:        models.BaseModel{ID: uuid.New()},
			TenantID:         tenantID,
			NeighborhoodName: neighborhood,
			City:             city,
			State:            state,
			ZoneType:         zoneType,
		}
		return s.db.Create(&zone).Error
	} else {
		return s.db.Where("tenant_id = ? AND LOWER(neighborhood_name) = LOWER(?) AND zone_type = ?",
			tenantID, neighborhood, zoneType).Delete(&models.TenantDeliveryZone{}).Error
	}
}

// GetDeliveryZones returns all delivery zones for a tenant
func (s *DeliveryService) GetDeliveryZones(tenantID uuid.UUID) ([]models.TenantDeliveryZone, error) {
	var zones []models.TenantDeliveryZone
	err := s.db.Where("tenant_id = ?", tenantID).Order("zone_type, neighborhood_name").Find(&zones).Error
	return zones, err
}

// GetTenantStoreConfiguration returns the store location configuration for a tenant
func (s *DeliveryService) GetTenantStoreConfiguration(tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := s.db.Select("id, store_street, store_number, store_neighborhood, store_city, store_state, store_zip_code, store_country, store_latitude, store_longitude, delivery_radius_km").
		First(&tenant, tenantID).Error
	return &tenant, err
}
