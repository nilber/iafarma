package models

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	BaseTenantModel
	CustomerID   uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"customer_id"`
	Label        string    `json:"label"` // home, work, etc.
	Name         string    `json:"name"`  // Added field that exists in table
	Street       string    `gorm:"not null" json:"street" validate:"required"`
	Number       string    `json:"number"`
	Complement   string    `json:"complement"`
	Neighborhood string    `json:"neighborhood"`
	City         string    `gorm:"not null" json:"city" validate:"required"`
	State        string    `gorm:"not null" json:"state" validate:"required"`
	ZipCode      string    `gorm:"column:zipcode;not null" json:"zip_code" validate:"required"` // Map to zipcode column
	Country      string    `gorm:"default:'BR'" json:"country"`
	IsDefault    bool      `gorm:"default:false" json:"is_default"`
}

type CreateAddressRequest struct {
	CustomerID   uuid.UUID `json:"customer_id" validate:"required"`
	Label        string    `json:"label"`
	Street       string    `json:"street" validate:"required"`
	Number       string    `json:"number"`
	Complement   string    `json:"complement"`
	Neighborhood string    `json:"neighborhood"`
	City         string    `json:"city" validate:"required"`
	State        string    `json:"state" validate:"required"`
	ZipCode      string    `json:"zip_code" validate:"required"`
	Country      string    `json:"country"`
	IsDefault    bool      `json:"is_default"`
}

type UpdateAddressRequest struct {
	Label        *string `json:"label"`
	Street       *string `json:"street"`
	Number       *string `json:"number"`
	Complement   *string `json:"complement"`
	Neighborhood *string `json:"neighborhood"`
	City         *string `json:"city"`
	State        *string `json:"state"`
	ZipCode      *string `json:"zip_code"`
	Country      *string `json:"country"`
	IsDefault    *bool   `json:"is_default"`
}

// MunicipioBrasileiro representa um município brasileiro
type MunicipioBrasileiro struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NomeCidade string    `gorm:"column:nome_cidade;not null" json:"nome_cidade"`
	UF         string    `gorm:"column:uf;not null" json:"uf"`
	Latitude   float64   `gorm:"column:latitude" json:"latitude"`
	Longitude  float64   `gorm:"column:longitude" json:"longitude"`
	DDD        int       `gorm:"column:ddd" json:"ddd"`
	Fuso       string    `gorm:"column:fuso" json:"fuso"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (MunicipioBrasileiro) TableName() string {
	return "municipios_brasileiros"
}
