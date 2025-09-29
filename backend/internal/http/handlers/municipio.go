package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type MunicipioHandler struct {
	db *gorm.DB
}

func NewMunicipioHandler(db *gorm.DB) *MunicipioHandler {
	return &MunicipioHandler{db: db}
}

// EstadoResponse representa um estado brasileiro
type EstadoResponse struct {
	UF   string `json:"uf"`
	Nome string `json:"nome"`
}

// CidadeResponse representa uma cidade brasileira
type CidadeResponse struct {
	ID         uint    `json:"id"`
	NomeCidade string  `json:"nome_cidade"`
	UF         string  `json:"uf"`
	DDD        int     `json:"ddd"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

// GetEstados godoc
// @Summary Get Brazilian states
// @Description Get list of Brazilian states from municipios_brasileiros table, distinct and ordered
// @Tags municipios
// @Accept json
// @Produce json
// @Param search query string false "Search term for state name or UF"
// @Success 200 {array} EstadoResponse
// @Failure 500 {object} map[string]string
// @Router /municipios/estados [get]
func (h *MunicipioHandler) GetEstados(c echo.Context) error {
	search := c.QueryParam("search")

	query := `
		SELECT DISTINCT uf,
		CASE uf 
			WHEN 'AC' THEN 'Acre'
			WHEN 'AL' THEN 'Alagoas'
			WHEN 'AP' THEN 'Amapá'
			WHEN 'AM' THEN 'Amazonas'
			WHEN 'BA' THEN 'Bahia'
			WHEN 'CE' THEN 'Ceará'
			WHEN 'DF' THEN 'Distrito Federal'
			WHEN 'ES' THEN 'Espírito Santo'
			WHEN 'GO' THEN 'Goiás'
			WHEN 'MA' THEN 'Maranhão'
			WHEN 'MT' THEN 'Mato Grosso'
			WHEN 'MS' THEN 'Mato Grosso do Sul'
			WHEN 'MG' THEN 'Minas Gerais'
			WHEN 'PA' THEN 'Pará'
			WHEN 'PB' THEN 'Paraíba'
			WHEN 'PR' THEN 'Paraná'
			WHEN 'PE' THEN 'Pernambuco'
			WHEN 'PI' THEN 'Piauí'
			WHEN 'RJ' THEN 'Rio de Janeiro'
			WHEN 'RN' THEN 'Rio Grande do Norte'
			WHEN 'RS' THEN 'Rio Grande do Sul'
			WHEN 'RO' THEN 'Rondônia'
			WHEN 'RR' THEN 'Roraima'
			WHEN 'SC' THEN 'Santa Catarina'
			WHEN 'SP' THEN 'São Paulo'
			WHEN 'SE' THEN 'Sergipe'
			WHEN 'TO' THEN 'Tocantins'
		END as nome
		FROM municipios_brasileiros
	`

	var args []interface{}
	if search != "" {
		searchLower := strings.ToLower(search)
		query += ` WHERE LOWER(uf) LIKE ? OR 
			CASE uf 
				WHEN 'AC' THEN 'acre'
				WHEN 'AL' THEN 'alagoas'
				WHEN 'AP' THEN 'amapá'
				WHEN 'AM' THEN 'amazonas'
				WHEN 'BA' THEN 'bahia'
				WHEN 'CE' THEN 'ceará'
				WHEN 'DF' THEN 'distrito federal'
				WHEN 'ES' THEN 'espírito santo'
				WHEN 'GO' THEN 'goiás'
				WHEN 'MA' THEN 'maranhão'
				WHEN 'MT' THEN 'mato grosso'
				WHEN 'MS' THEN 'mato grosso do sul'
				WHEN 'MG' THEN 'minas gerais'
				WHEN 'PA' THEN 'pará'
				WHEN 'PB' THEN 'paraíba'
				WHEN 'PR' THEN 'paraná'
				WHEN 'PE' THEN 'pernambuco'
				WHEN 'PI' THEN 'piauí'
				WHEN 'RJ' THEN 'rio de janeiro'
				WHEN 'RN' THEN 'rio grande do norte'
				WHEN 'RS' THEN 'rio grande do sul'
				WHEN 'RO' THEN 'rondônia'
				WHEN 'RR' THEN 'roraima'
				WHEN 'SC' THEN 'santa catarina'
				WHEN 'SP' THEN 'são paulo'
				WHEN 'SE' THEN 'sergipe'
				WHEN 'TO' THEN 'tocantins'
			END LIKE ?`
		args = append(args, "%"+searchLower+"%", "%"+searchLower+"%")
	}

	query += " ORDER BY uf ASC"

	var estados []EstadoResponse
	if err := h.db.Raw(query, args...).Scan(&estados).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch states",
		})
	}

	return c.JSON(http.StatusOK, estados)
}

// GetCidades godoc
// @Summary Get Brazilian cities by state
// @Description Get list of Brazilian cities from municipios_brasileiros table filtered by state
// @Tags municipios
// @Accept json
// @Produce json
// @Param uf query string true "State UF code (e.g., SP, RJ)"
// @Param search query string false "Search term for city name"
// @Success 200 {array} CidadeResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /municipios/cidades [get]
func (h *MunicipioHandler) GetCidades(c echo.Context) error {
	uf := c.QueryParam("uf")
	if uf == "" {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"error": "UF parameter is required",
		})
	}

	search := c.QueryParam("search")

	query := `
		SELECT id, nome_cidade, uf, ddd, latitude, longitude
		FROM municipios_brasileiros 
		WHERE uf = ?
	`
	args := []interface{}{strings.ToUpper(uf)}

	if search != "" {
		query += " AND LOWER(nome_cidade) LIKE ?"
		args = append(args, "%"+strings.ToLower(search)+"%")
	}

	query += " ORDER BY nome_cidade ASC"

	var cidades []CidadeResponse
	if err := h.db.Raw(query, args...).Scan(&cidades).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch cities",
		})
	}

	return c.JSON(http.StatusOK, cidades)
}
