package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"iafarma/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ImportJobHandler struct {
	importJobService *services.ImportJobService
}

func NewImportJobHandler(importJobService *services.ImportJobService) *ImportJobHandler {
	return &ImportJobHandler{
		importJobService: importJobService,
	}
}

// CreateProductImportJob inicia um job de importação de produtos
// @Summary Create product import job
// @Description Start a new asynchronous product import job
// @Tags import-jobs
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file to import"
// @Success 202 {object} models.ImportJob
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /import/products [post]
func (h *ImportJobHandler) CreateProductImportJob(c echo.Context) error {
	// Debug: log all context values
	c.Logger().Info("Context keys: ", c.Get("tenant_id"), c.Get("user_id"))

	tenantID, exists := c.Get("tenant_id").(uuid.UUID)
	if !exists {
		c.Logger().Error("tenant_id not found in context or wrong type")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found"})
	}

	userID, exists := c.Get("user_id").(uuid.UUID)
	if !exists {
		c.Logger().Error("user_id not found in context or wrong type")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found"})
	}

	// Obter arquivo do formulário
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "arquivo não encontrado"})
	}
	defer file.Close()

	// Validar tipo de arquivo
	if header.Header.Get("Content-Type") != "text/csv" && !strings.HasSuffix(header.Filename, ".csv") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "apenas arquivos CSV são aceitos"})
	}

	// Criar job de importação
	job, err := h.importJobService.CreateProductImportJob(c.Request().Context(), tenantID, userID, file, header)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Retornar resposta no formato esperado pelo frontend
	response := map[string]interface{}{
		"job_id":  job.ID.String(),
		"message": "Importação iniciada com sucesso",
	}

	return c.JSON(http.StatusAccepted, response)
}

// GetImportJobProgress retorna o progresso de um job de importação
// @Summary Get import job progress
// @Description Get the progress of an import job
// @Tags import-jobs
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} models.ImportJobProgress
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /import/jobs/{job_id}/progress [get]
func (h *ImportJobHandler) GetImportJobProgress(c echo.Context) error {
	tenantID, exists := c.Get("tenant_id").(uuid.UUID)
	if !exists {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found"})
	}

	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "job_id inválido"})
	}

	progress, err := h.importJobService.GetJobProgress(c.Request().Context(), tenantID, jobID)
	if err != nil {
		if err.Error() == "job not found: record not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "job não encontrado"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Adaptar resposta para o formato esperado pelo frontend
	response := map[string]interface{}{
		"job": map[string]interface{}{
			"id":               progress.JobID.String(),
			"status":           progress.Status,
			"total_items":      progress.TotalRecords,
			"processed_items":  progress.ProcessedRecords,
			"successful_items": progress.SuccessRecords,
			"failed_items":     progress.ErrorRecords,
			"error_message": func() *string {
				if len(progress.ErrorDetails) > 0 {
					msg := strings.Join(progress.ErrorDetails, "; ")
					return &msg
				}
				return nil
			}(),
		},
		"progress_percentage": progress.Progress,
	}

	return c.JSON(http.StatusOK, response)
}

// ListImportJobs lista jobs de importação do tenant
// @Summary List import jobs
// @Description List all import jobs for the current tenant
// @Tags import-jobs
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status"
// @Success 200 {object} []models.ImportJob
// @Failure 500 {object} map[string]string
// @Router /import/jobs [get]
func (h *ImportJobHandler) ListImportJobs(c echo.Context) error {
	tenantID, exists := c.Get("tenant_id").(uuid.UUID)
	if !exists {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found"})
	}

	// Parse query parameters
	page := 1
	if p := c.QueryParam("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 10
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	status := c.QueryParam("status")

	jobs, total, err := h.importJobService.ListJobs(c.Request().Context(), tenantID, page, limit, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	response := map[string]interface{}{
		"jobs":  jobs,
		"total": total,
		"page":  page,
		"limit": limit,
	}

	return c.JSON(http.StatusOK, response)
}
