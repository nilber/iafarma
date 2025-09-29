package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iafarma/internal/repo"
	"iafarma/internal/utils"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ImportJobService struct {
	db               *gorm.DB
	productRepo      *repo.ProductRepository
	categoryRepo     *repo.CategoryRepository
	embeddingService *EmbeddingService
	uploadDir        string
}

func NewImportJobService(db *gorm.DB, productRepo *repo.ProductRepository, categoryRepo *repo.CategoryRepository, embeddingService *EmbeddingService) *ImportJobService {
	uploadDir := "uploads/imports"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Error creating upload directory: %v", err)
	}

	return &ImportJobService{
		db:               db,
		productRepo:      productRepo,
		categoryRepo:     categoryRepo,
		embeddingService: embeddingService,
		uploadDir:        uploadDir,
	}
}

// CreateProductImportJob cria um job de importação de produtos
func (s *ImportJobService) CreateProductImportJob(ctx context.Context, tenantID, userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*models.ImportJob, error) {
	// Salvar arquivo temporário
	fileName := fmt.Sprintf("%s_%s", uuid.New().String(), header.Filename)
	filePath := filepath.Join(s.uploadDir, fileName)

	outFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Copiar conteúdo do arquivo
	_, err = io.Copy(outFile, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Contar registros no CSV
	totalRecords, err := s.countCSVRecords(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// Criar job no banco
	job := &models.ImportJob{
		TenantID:     tenantID,
		UserID:       userID,
		Type:         models.ImportJobTypeProducts,
		Status:       models.ImportJobStatusPending,
		FileName:     header.Filename,
		FilePath:     filePath,
		TotalRecords: totalRecords,
	}

	err = s.db.WithContext(ctx).Create(job).Error
	if err != nil {
		// Limpar arquivo se falhou ao criar job
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	// Iniciar processamento em goroutine separada
	go s.processProductImportJob(job.ID)

	return job, nil
}

// GetJobProgress retorna o progresso de um job
func (s *ImportJobService) GetJobProgress(ctx context.Context, tenantID, jobID uuid.UUID) (*models.ImportJobProgress, error) {
	var job models.ImportJob
	err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	progress := job.ToProgress()

	// Adicionar mensagem baseada no status
	switch job.Status {
	case models.ImportJobStatusPending:
		progress.Message = "Aguardando processamento..."
	case models.ImportJobStatusProcessing:
		progress.Message = fmt.Sprintf("Processando %d de %d registros...", job.ProcessedRecords, job.TotalRecords)
	case models.ImportJobStatusCompleted:
		progress.Message = fmt.Sprintf("Concluído! %d criados, %d com erro", job.SuccessRecords, job.ErrorRecords)
	case models.ImportJobStatusFailed:
		progress.Message = "Falha no processamento"
	}

	// Adicionar detalhes de erro se houver
	if job.ErrorDetails != nil && *job.ErrorDetails != "" {
		var errorDetails []string
		if err := json.Unmarshal([]byte(*job.ErrorDetails), &errorDetails); err == nil {
			progress.ErrorDetails = errorDetails
		}
	}

	return &progress, nil
}

// countCSVRecords conta o número de registros em um arquivo CSV
func (s *ImportJobService) countCSVRecords(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Usar detecção automática de delimitador para contagem
	records, _, err := utils.ParseCSVWithDetectedDelimiter(file)
	if err != nil {
		return 0, err
	}

	// Subtrair 1 para o cabeçalho
	if len(records) > 0 {
		return len(records) - 1, nil
	}
	return 0, nil
}

// processProductImportJob processa um job de importação de produtos
func (s *ImportJobService) processProductImportJob(jobID uuid.UUID) {
	ctx := context.Background()

	// Buscar job
	var job models.ImportJob
	err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error
	if err != nil {
		log.Printf("Failed to find import job %s: %v", jobID, err)
		return
	}

	// Marcar como iniciado
	now := time.Now()
	job.Status = models.ImportJobStatusProcessing
	job.StartedAt = &now
	s.db.WithContext(ctx).Save(&job)

	// Processar arquivo
	err = s.processProductsCSV(ctx, &job)

	// Marcar como concluído
	completedAt := time.Now()
	job.CompletedAt = &completedAt

	if err != nil {
		job.Status = models.ImportJobStatusFailed
		errorDetails, _ := json.Marshal([]string{err.Error()})
		errorString := string(errorDetails)
		job.ErrorDetails = &errorString
		log.Printf("Import job %s failed: %v", jobID, err)
	} else {
		job.Status = models.ImportJobStatusCompleted
		log.Printf("Import job %s completed: %d processed, %d success, %d errors",
			jobID, job.ProcessedRecords, job.SuccessRecords, job.ErrorRecords)
	}

	s.db.WithContext(ctx).Save(&job)

	// Limpar arquivo após processamento
	os.Remove(job.FilePath)
}

// processProductsCSV processa o arquivo CSV de produtos
func (s *ImportJobService) processProductsCSV(ctx context.Context, job *models.ImportJob) error {
	file, err := os.Open(job.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Usar detecção automática de delimitador
	records, analysis, err := utils.ParseCSVWithDetectedDelimiter(file)
	if err != nil {
		return fmt.Errorf("failed to parse CSV with auto-detection: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file must have at least header and one data row")
	}

	log.Printf("CSV Analysis - Delimiter: %c, Numeric Separator: %s, Confidence: %.2f",
		analysis.Delimiter, analysis.NumericSeparator, analysis.DelimiterConfidence)

	headers := records[0]
	headerMap := make(map[string]int)
	for i, header := range headers {
		// Limpar e normalizar header
		cleanHeader := strings.ToLower(strings.TrimSpace(header))
		cleanHeader = strings.Trim(cleanHeader, `"'`)
		headerMap[cleanHeader] = i
	}

	totalRecords := len(records) - 1 // Excluir header

	// Acumular produtos para processamento em lote
	var productsForBatch []models.Product

	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]

		// Criar produto a partir do CSV
		product, err := s.parseProductFromCSVWithAnalysis(record, headerMap, job.TenantID, analysis)
		if err != nil {
			log.Printf("Error parsing product at row %d: %v", rowIndex+1, err)
			job.ErrorRecords++
			job.ProcessedRecords++
			continue
		}

		// Salvar produto
		err = s.createOrUpdateProductWithoutRAG(ctx, product)
		if err != nil {
			log.Printf("Error saving product at row %d: %v", rowIndex+1, err)
			job.ErrorRecords++
		} else {
			job.SuccessRecords++
			// Adicionar à lista para processamento em lote do RAG
			productsForBatch = append(productsForBatch, *product)
		}

		job.ProcessedRecords++

		// Atualizar progresso no banco a cada 100 registros
		if job.ProcessedRecords%100 == 0 || job.ProcessedRecords == totalRecords {
			log.Printf("Import progress: %d/%d products processed", job.ProcessedRecords, totalRecords)

			// Salvar progresso no banco
			err := s.db.WithContext(ctx).Model(&job).Updates(map[string]interface{}{
				"processed_records": job.ProcessedRecords,
				"success_records":   job.SuccessRecords,
				"error_records":     job.ErrorRecords,
			}).Error
			if err != nil {
				log.Printf("Error updating job progress: %v", err)
			}
		}
	}

	// Processar embeddings em lote após salvar todos os produtos
	if s.embeddingService != nil && len(productsForBatch) > 0 {
		log.Printf("🔄 Starting batch embedding processing for %d imported products (tenant: %s) - embeddingService available", len(productsForBatch), job.TenantID.String())

		// Recarregar produtos do banco para ter os embedding_hash atualizados
		var reloadedProducts []models.Product
		var productIDs []uuid.UUID
		for _, p := range productsForBatch {
			productIDs = append(productIDs, p.ID)
		}

		err := s.db.Select("*").Where("id IN ?", productIDs).Find(&reloadedProducts).Error
		if err != nil {
			log.Printf("Failed to reload products for hash verification: %v", err)
			// Fallback to original products
			reloadedProducts = productsForBatch
		}

		// Usar processamento assíncrono para não bloquear a finalização do job
		go func() {
			if err := s.processBatchEmbeddings(reloadedProducts, job.TenantID); err != nil {
				log.Printf("❌ Failed to process batch embeddings for import job %s: %v", job.ID.String(), err)
			}
		}()
	} else if s.embeddingService == nil {
		log.Printf("⚠️  Embedding service is nil - skipping batch embedding processing")
	} else if len(productsForBatch) == 0 {
		log.Printf("ℹ️  No products to process embeddings for in batch")
	}

	log.Printf("Import completed: %d/%d products processed", job.SuccessRecords, totalRecords)
	return nil
}

// parseProductFromCSVWithAnalysis converte uma linha CSV em produto com análise de formato
func (s *ImportJobService) parseProductFromCSVWithAnalysis(record []string, headerMap map[string]int, tenantID uuid.UUID, analysis *utils.CSVAnalysisResult) (*models.Product, error) {
	getValue := func(header string) string {
		if idx, exists := headerMap[header]; exists && idx < len(record) {
			value := strings.TrimSpace(record[idx])
			value = strings.Trim(value, `"'`)
			return value
		}
		return ""
	}

	getNumericValue := func(header string) string {
		value := getValue(header)
		if value == "" {
			return value
		}
		// Normalizar valor numérico baseado na análise
		return utils.NormalizeNumericValue(value, analysis.NumericSeparator)
	}

	name := getValue("name")
	if name == "" && getValue("nome") != "" {
		name = getValue("nome")
	}
	if name == "" {
		return nil, fmt.Errorf("nome do produto é obrigatório")
	}

	priceStr := getNumericValue("price")
	if priceStr == "" && getNumericValue("preco") != "" {
		priceStr = getNumericValue("preco")
	}
	if priceStr == "" && getNumericValue("preço") != "" {
		priceStr = getNumericValue("preço")
	}
	if priceStr == "" {
		return nil, fmt.Errorf("preço é obrigatório")
	}

	product := &models.Product{}
	product.TenantID = tenantID
	product.Name = name
	product.Description = getValue("description")
	if product.Description == "" {
		product.Description = getValue("descricao")
	}
	if product.Description == "" {
		product.Description = getValue("descrição")
	}

	product.Price = priceStr
	product.SKU = getValue("sku")
	product.Barcode = getValue("barcode")
	if product.Barcode == "" {
		product.Barcode = getValue("codigo_barra")
	}

	product.Weight = getValue("weight")
	if product.Weight == "" {
		product.Weight = getValue("peso")
	}

	product.Dimensions = getValue("dimensions")
	if product.Dimensions == "" {
		product.Dimensions = getValue("dimensoes")
	}

	product.Brand = getValue("brand")
	if product.Brand == "" {
		product.Brand = getValue("marca")
	}

	product.Tags = getValue("tags")

	// Campos opcionais com normalização numérica
	if salePriceStr := getNumericValue("sale_price"); salePriceStr != "" {
		product.SalePrice = salePriceStr
	} else if salePriceStr := getNumericValue("preco_promocional"); salePriceStr != "" {
		product.SalePrice = salePriceStr
	}

	if stockStr := getValue("stock_quantity"); stockStr != "" {
		if stock, err := strconv.Atoi(stockStr); err == nil {
			product.StockQuantity = stock
		}
	} else if stockStr := getValue("estoque"); stockStr != "" {
		if stock, err := strconv.Atoi(stockStr); err == nil {
			product.StockQuantity = stock
		}
	}

	if lowStockStr := getValue("low_stock_threshold"); lowStockStr != "" {
		if lowStock, err := strconv.Atoi(lowStockStr); err == nil {
			product.LowStockThreshold = lowStock
		}
	} else if lowStockStr := getValue("estoque_minimo"); lowStockStr != "" {
		if lowStock, err := strconv.Atoi(lowStockStr); err == nil {
			product.LowStockThreshold = lowStock
		}
	}

	// Tratamento de categoria - buscar ou criar se fornecida
	categoryName := getValue("category_name")
	if categoryName == "" {
		categoryName = getValue("categoria")
	}
	if categoryName == "" {
		categoryName = getValue("category")
	}

	if categoryName != "" {
		categoryID, err := s.findOrCreateCategory(tenantID, categoryName)
		if err != nil {
			log.Printf("Warning: Failed to find/create category '%s': %v", categoryName, err)
		} else {
			product.CategoryID = categoryID
		}
	}

	return product, nil
}

// findOrCreateCategory finds an existing category by name or creates a new one
func (s *ImportJobService) findOrCreateCategory(tenantID uuid.UUID, categoryName string) (*uuid.UUID, error) {
	if categoryName == "" {
		return nil, nil
	}

	// Try to find existing category
	existing, err := s.categoryRepo.FindExistingCategory(tenantID, categoryName)
	if err == nil && existing != nil {
		return &existing.ID, nil
	}

	// If not found, create new category
	if err == gorm.ErrRecordNotFound {
		category := &models.Category{
			BaseTenantModel: models.BaseTenantModel{
				TenantID: tenantID,
			},
			Name:        categoryName,
			Description: "",
			IsActive:    true,
			SortOrder:   0,
		}

		if createErr := s.categoryRepo.Create(category); createErr != nil {
			return nil, createErr
		}

		return &category.ID, nil
	}

	// Return other errors
	return nil, err
}

// createOrUpdateProduct cria ou atualiza um produto
func (s *ImportJobService) createOrUpdateProduct(ctx context.Context, product *models.Product) error {
	// Verificar se produto já existe pelo SKU ou nome
	var existingProduct models.Product
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND (sku = ? OR name = ?)",
		product.TenantID, product.SKU, product.Name).First(&existingProduct).Error

	var isNewProduct bool
	if err == gorm.ErrRecordNotFound {
		// Criar novo produto
		isNewProduct = true
		err = s.db.WithContext(ctx).Create(product).Error
	} else if err != nil {
		return fmt.Errorf("erro ao verificar produto existente: %w", err)
	} else {
		// Atualizar produto existente
		product.ID = existingProduct.ID
		product.CreatedAt = existingProduct.CreatedAt
		err = s.db.WithContext(ctx).Save(product).Error
	}

	if err != nil {
		return err
	}

	// Registrar produto no RAG se o embedding service estiver disponível
	if s.embeddingService != nil {
		go func() {
			searchText := product.GetSearchText()
			metadata := product.GetMetadata()
			if err := s.embeddingService.StoreProductEmbedding(
				product.ID.String(),
				product.TenantID.String(),
				searchText,
				metadata,
			); err != nil {
				log.Printf("Failed to store embedding for product %s: %v", product.ID, err)
			} else {
				if isNewProduct {
					log.Printf("Product embedding stored successfully for product %s in tenant %s", product.ID, product.TenantID)
				} else {
					log.Printf("Product embedding updated successfully for product %s in tenant %s", product.ID, product.TenantID)
				}
			}
		}()
	}

	return nil
}

// createOrUpdateProductWithoutRAG cria ou atualiza um produto sem processar embeddings
func (s *ImportJobService) createOrUpdateProductWithoutRAG(ctx context.Context, product *models.Product) error {
	// Usar o mesmo método UpsertProduct que a importação regular usa
	savedProduct, _, err := s.productRepo.UpsertProduct(product)
	if err != nil {
		return err
	}

	// Atualizar o produto com os dados salvos (como ID, created_at, etc.)
	*product = *savedProduct
	return nil
}

// processBatchEmbeddings processa embeddings em lote para evitar rate limits
func (s *ImportJobService) processBatchEmbeddings(products []models.Product, tenantID uuid.UUID) error {
	if s.embeddingService == nil {
		return nil
	}

	// Pre-filter products that need processing (hash changed or missing)
	var productsToProcess []models.Product
	var skippedCount int

	for _, product := range products {
		searchText := product.GetSearchText()
		currentHash := s.calculateContentHash(searchText)

		// Check if product needs reprocessing
		if product.EmbeddingHash != "" && product.EmbeddingHash == currentHash {
			skippedCount++
			continue
		}

		productsToProcess = append(productsToProcess, product)
	}

	log.Printf("📊 Import batch analysis: %d products to process, %d skipped (hash unchanged)",
		len(productsToProcess), skippedCount)

	if len(productsToProcess) == 0 {
		log.Printf("✅ Import batch complete - all products up to date")
		return nil
	}

	batchSize := 500 // Optimized batch size for better performance

	for i := 0; i < len(productsToProcess); i += batchSize {
		end := i + batchSize
		if end > len(productsToProcess) {
			end = len(productsToProcess)
		}

		batch := productsToProcess[i:end]

		// Preparar dados para o lote usando BatchProductData
		var batchProducts []BatchProductData
		var productHashes []struct {
			ProductID uuid.UUID
			Hash      string
		}

		for _, product := range batch {
			searchText := product.GetSearchText()
			hash := s.calculateContentHash(searchText)

			batchProduct := BatchProductData{
				ID:       product.ID.String(),
				TenantID: tenantID.String(),
				Text:     searchText,
				Metadata: product.GetMetadata(),
			}
			batchProducts = append(batchProducts, batchProduct)

			productHashes = append(productHashes, struct {
				ProductID uuid.UUID
				Hash      string
			}{
				ProductID: product.ID,
				Hash:      hash,
			})
		}

		// Processar lote
		err := s.embeddingService.StoreBatchProductEmbeddings(
			tenantID.String(),
			batchProducts,
			len(batchProducts),
		)

		if err != nil {
			log.Printf("❌ Failed to process batch embeddings for batch %d-%d (tenant: %s): %v", i, end-1, tenantID, err)
		} else {
			log.Printf("✅ Successfully processed batch embeddings for products %d-%d (tenant: %s)", i, end-1, tenantID)

			// Update hashes in database after successful embedding storage
			for _, productHash := range productHashes {
				s.updateProductEmbeddingHash(productHash.ProductID, productHash.Hash)
			}
		}

		// Pequeno delay entre lotes para evitar rate limits
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// calculateContentHash generates a hash of the product content for caching
func (s *ImportJobService) calculateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars
}

// updateProductEmbeddingHash updates the embedding hash in the database
func (s *ImportJobService) updateProductEmbeddingHash(productID uuid.UUID, hash string) {
	if err := s.db.Model(&models.Product{}).Where("id = ?", productID).Update("embedding_hash", hash).Error; err != nil {
		log.Printf("Failed to update embedding hash for product %s: %v", productID, err)
	} else {
		log.Printf("✅ Updated embedding hash for product %s to: %s", productID, hash)
	}
}

// ListJobs lista os jobs de importação com paginação e filtros
func (s *ImportJobService) ListJobs(ctx context.Context, tenantID uuid.UUID, page, limit int, status string) ([]*models.ImportJob, int64, error) {
	var jobs []*models.ImportJob
	var total int64

	query := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Filtrar por status se fornecido
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Contar total de registros
	if err := query.Model(&models.ImportJob{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao contar jobs: %w", err)
	}

	// Aplicar paginação e ordenação
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar jobs: %w", err)
	}

	return jobs, total, nil
}
