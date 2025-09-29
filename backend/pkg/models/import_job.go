package models

import (
	"time"

	"github.com/google/uuid"
)

type ImportJobStatus string

const (
	ImportJobStatusPending    ImportJobStatus = "pending"
	ImportJobStatusProcessing ImportJobStatus = "processing"
	ImportJobStatusCompleted  ImportJobStatus = "completed"
	ImportJobStatusFailed     ImportJobStatus = "failed"
)

type ImportJobType string

const (
	ImportJobTypeProducts ImportJobType = "products"
)

// ImportJob representa um job de importação assíncrona
type ImportJob struct {
	BaseModel
	TenantID         uuid.UUID       `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	UserID           uuid.UUID       `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"user_id"`
	Type             ImportJobType   `gorm:"not null" json:"type"`
	Status           ImportJobStatus `gorm:"not null;default:'pending'" json:"status"`
	FileName         string          `gorm:"not null" json:"file_name"`
	FilePath         string          `gorm:"not null" json:"file_path"`
	TotalRecords     int             `gorm:"default:0" json:"total_records"`
	ProcessedRecords int             `gorm:"default:0" json:"processed_records"`
	SuccessRecords   int             `gorm:"default:0" json:"success_records"`
	ErrorRecords     int             `gorm:"default:0" json:"error_records"`
	ErrorDetails     *string         `gorm:"type:jsonb" json:"error_details,omitempty"`
	Result           *string         `gorm:"type:jsonb" json:"result,omitempty"`
	StartedAt        *time.Time      `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at"`
}

// ImportJobProgress representa o progresso de um job
type ImportJobProgress struct {
	JobID            uuid.UUID       `json:"job_id"`
	Status           ImportJobStatus `json:"status"`
	TotalRecords     int             `json:"total_records"`
	ProcessedRecords int             `json:"processed_records"`
	SuccessRecords   int             `json:"success_records"`
	ErrorRecords     int             `json:"error_records"`
	Progress         float64         `json:"progress"` // 0-100
	Message          string          `json:"message"`
	ErrorDetails     []string        `json:"error_details,omitempty"`
}

// CalculateProgress calcula o progresso percentual
func (job *ImportJob) CalculateProgress() float64 {
	if job.TotalRecords == 0 {
		return 0
	}
	return float64(job.ProcessedRecords) / float64(job.TotalRecords) * 100
}

// ToProgress converte para estrutura de progresso
func (job *ImportJob) ToProgress() ImportJobProgress {
	return ImportJobProgress{
		JobID:            job.ID,
		Status:           job.Status,
		TotalRecords:     job.TotalRecords,
		ProcessedRecords: job.ProcessedRecords,
		SuccessRecords:   job.SuccessRecords,
		ErrorRecords:     job.ErrorRecords,
		Progress:         job.CalculateProgress(),
		Message:          "Processando importação...",
	}
}
