package ai

import (
	"context"
	"fmt"
	"iafarma/pkg/models"
	"reflect"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// DeliveryServiceAdapter adapts the real DeliveryService to the AI interface
type DeliveryServiceAdapter struct {
	service interface{}
}

// NewDeliveryServiceAdapter creates a new adapter
func NewDeliveryServiceAdapter(service interface{}) *DeliveryServiceAdapter {
	return &DeliveryServiceAdapter{
		service: service,
	}
}

// ValidateDeliveryAddress validates if delivery is possible to the given address
func (a *DeliveryServiceAdapter) ValidateDeliveryAddress(tenantID uuid.UUID, street, number, neighborhood, city, state string) (*DeliveryValidationResult, error) {
	// Create address from parameters
	address := models.Address{
		Street:       street,
		Number:       number,
		Neighborhood: neighborhood,
		City:         city,
		State:        state,
	}

	ctx := context.Background()

	// Use reflection to call the ValidateDeliveryAddress method
	serviceValue := reflect.ValueOf(a.service)
	method := serviceValue.MethodByName("ValidateDeliveryAddress")
	if !method.IsValid() {
		return nil, fmt.Errorf("ValidateDeliveryAddress method not found")
	}

	// Call the method with context, tenantID, and address
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(tenantID),
		reflect.ValueOf(address),
	})

	// Check for error
	if len(results) != 2 {
		return nil, fmt.Errorf("unexpected number of return values")
	}

	// Extract error
	if !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, err
		}
	}

	// Extract result and convert using reflection
	resultInterface := results[0].Interface()
	result := &DeliveryValidationResult{
		CanDeliver: false,
		Reason:     "Não foi possível validar a entrega",
		ZoneType:   "unknown",
		Distance:   "",
	}

	if resultInterface != nil {
		v := reflect.ValueOf(resultInterface)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Struct {
			// Extract CanDeliver
			if canDeliverField := v.FieldByName("CanDeliver"); canDeliverField.IsValid() && canDeliverField.CanInterface() {
				if canDeliver, ok := canDeliverField.Interface().(bool); ok {
					result.CanDeliver = canDeliver
				}
			}

			// Extract Reason
			if reasonField := v.FieldByName("Reason"); reasonField.IsValid() && reasonField.CanInterface() {
				if reason, ok := reasonField.Interface().(string); ok {
					result.Reason = reason
				}
			}

			// Extract ZoneType
			if zoneTypeField := v.FieldByName("ZoneType"); zoneTypeField.IsValid() && zoneTypeField.CanInterface() {
				if zoneType, ok := zoneTypeField.Interface().(string); ok {
					result.ZoneType = zoneType
				}
			}

			// Extract Distance
			if distanceField := v.FieldByName("Distance"); distanceField.IsValid() && distanceField.CanInterface() {
				if distance, ok := distanceField.Interface().(string); ok {
					result.Distance = distance
				}
			}

			// Extract DistanceKm for detailed logging
			var distanceKm float64
			if distanceKmField := v.FieldByName("DistanceKm"); distanceKmField.IsValid() && distanceKmField.CanInterface() {
				if distKm, ok := distanceKmField.Interface().(float64); ok {
					distanceKm = distKm
				}
			}

			// Log detailed delivery validation result
			log.Info().
				Str("tenant_id", tenantID.String()).
				Str("neighborhood", neighborhood).
				Str("city", city).
				Str("state", state).
				Bool("can_deliver", result.CanDeliver).
				Str("reason", result.Reason).
				Str("zone_type", result.ZoneType).
				Str("distance", result.Distance).
				Float64("distance_km", distanceKm).
				Msg("🚚 DELIVERY VALIDATION RESULT - Detailed distance calculation")
		}
	}

	return result, nil
}

// GetStoreLocation gets the store location information
func (a *DeliveryServiceAdapter) GetStoreLocation(tenantID uuid.UUID) (*StoreLocationInfo, error) {
	// Use reflection to call the GetTenantStoreConfiguration method
	serviceValue := reflect.ValueOf(a.service)
	method := serviceValue.MethodByName("GetTenantStoreConfiguration")
	if !method.IsValid() {
		return nil, fmt.Errorf("GetTenantStoreConfiguration method not found")
	}

	// Call the method with tenantID
	results := method.Call([]reflect.Value{
		reflect.ValueOf(tenantID),
	})

	// Check for error
	if len(results) != 2 {
		return nil, fmt.Errorf("unexpected number of return values from GetTenantStoreConfiguration")
	}

	// Extract error
	if !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, err
		}
	}

	// Extract tenant
	tenantInterface := results[0].Interface()
	tenant, ok := tenantInterface.(*models.Tenant)
	if !ok {
		return nil, fmt.Errorf("unexpected return type from GetTenantStoreConfiguration")
	}

	// Build address string
	addressParts := []string{}
	if tenant.StoreStreet != "" {
		addressParts = append(addressParts, tenant.StoreStreet)
	}
	if tenant.StoreNumber != "" {
		addressParts = append(addressParts, tenant.StoreNumber)
	}
	if tenant.StoreNeighborhood != "" {
		addressParts = append(addressParts, tenant.StoreNeighborhood)
	}
	if tenant.StoreCity != "" {
		addressParts = append(addressParts, tenant.StoreCity)
	}
	if tenant.StoreState != "" {
		addressParts = append(addressParts, tenant.StoreState)
	}

	address := ""
	if len(addressParts) > 0 {
		address = addressParts[0]
		for _, part := range addressParts[1:] {
			address += ", " + part
		}
	}

	latitude := 0.0
	longitude := 0.0
	if tenant.StoreLatitude != nil {
		latitude = *tenant.StoreLatitude
	}
	if tenant.StoreLongitude != nil {
		longitude = *tenant.StoreLongitude
	}

	return &StoreLocationInfo{
		Address:     address,
		City:        tenant.StoreCity,
		State:       tenant.StoreState,
		Coordinates: tenant.StoreLatitude != nil && tenant.StoreLongitude != nil && *tenant.StoreLatitude != 0 && *tenant.StoreLongitude != 0,
		Latitude:    latitude,
		Longitude:   longitude,
		RadiusKm:    float64(tenant.DeliveryRadiusKm),
	}, nil
}
