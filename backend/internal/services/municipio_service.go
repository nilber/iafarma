package services

import (
	"encoding/csv"
	"fmt"
	"iafarma/pkg/models"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type MunicipioService struct {
	db *gorm.DB
}

func NewMunicipioService(db *gorm.DB) *MunicipioService {
	return &MunicipioService{db: db}
}

// ImportarMunicipiosFromCSV importa os municípios do arquivo CSV
func (s *MunicipioService) ImportarMunicipiosFromCSV(csvPath string) error {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("erro ao ler CSV: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("arquivo CSV vazio")
	}

	// Pular header (primeira linha)
	records = records[1:]

	// Processar em lotes para performance
	batchSize := 1000
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		municipios := make([]models.MunicipioBrasileiro, 0, len(batch))

		for _, record := range batch {
			if len(record) < 6 {
				continue // Pular linhas incompletas
			}

			latitude, _ := strconv.ParseFloat(record[2], 64)
			longitude, _ := strconv.ParseFloat(record[3], 64)
			ddd, _ := strconv.Atoi(record[4])

			municipio := models.MunicipioBrasileiro{
				NomeCidade: strings.TrimSpace(record[0]),
				UF:         strings.TrimSpace(record[1]),
				Latitude:   latitude,
				Longitude:  longitude,
				DDD:        ddd,
				Fuso:       strings.TrimSpace(record[5]),
			}
			municipios = append(municipios, municipio)
		}

		if len(municipios) > 0 {
			// Usar ON CONFLICT para evitar duplicatas
			if err := s.db.Create(&municipios).Error; err != nil {
				return fmt.Errorf("erro ao inserir lote %d-%d: %w", i, end, err)
			}
		}
	}

	return nil
}

// ValidarCidade verifica se uma cidade existe no banco de dados
func (s *MunicipioService) ValidarCidade(nomeCidade, uf string) (bool, *models.MunicipioBrasileiro, error) {
	if nomeCidade == "" {
		return false, nil, nil
	}

	// Normalizar entrada
	nomeNormalizado := s.normalizeText(nomeCidade)
	ufNormalizado := strings.ToUpper(strings.TrimSpace(uf))

	var municipio models.MunicipioBrasileiro

	// Buscar exato primeiro
	err := s.db.Raw(`
		SELECT * FROM municipios_brasileiros 
		WHERE normalize_text(nome_cidade) = ? AND uf = ?
		LIMIT 1
	`, nomeNormalizado, ufNormalizado).First(&municipio).Error

	if err == nil {
		return true, &municipio, nil
	}

	if err != gorm.ErrRecordNotFound {
		return false, nil, err
	}

	// Se não encontrou, tentar busca mais flexível (LIKE)
	err = s.db.Raw(`
		SELECT * FROM municipios_brasileiros 
		WHERE normalize_text(nome_cidade) LIKE ? AND uf = ?
		LIMIT 1
	`, "%"+nomeNormalizado+"%", ufNormalizado).First(&municipio).Error

	if err == nil {
		return true, &municipio, nil
	}

	if err == gorm.ErrRecordNotFound {
		return false, nil, nil
	}

	return false, nil, err
}

// BuscarCidadesPorUF retorna todas as cidades de uma UF
func (s *MunicipioService) BuscarCidadesPorUF(uf string) ([]models.MunicipioBrasileiro, error) {
	var municipios []models.MunicipioBrasileiro

	err := s.db.Where("uf = ?", strings.ToUpper(strings.TrimSpace(uf))).
		Order("nome_cidade").
		Find(&municipios).Error

	return municipios, err
}

// BuscarCidadesComSimilaridade busca cidades similares ao nome fornecido
func (s *MunicipioService) BuscarCidadesComSimilaridade(nomeCidade, uf string, limite int) ([]models.MunicipioBrasileiro, error) {
	if limite <= 0 {
		limite = 5
	}

	nomeNormalizado := s.normalizeText(nomeCidade)
	ufNormalizado := strings.ToUpper(strings.TrimSpace(uf))

	var municipios []models.MunicipioBrasileiro

	query := s.db.Raw(`
		SELECT *, 
		CASE 
			WHEN normalize_text(nome_cidade) = ? THEN 1
			WHEN normalize_text(nome_cidade) LIKE ? THEN 2
			WHEN normalize_text(nome_cidade) LIKE ? THEN 3
			ELSE 4
		END as similarity_rank
		FROM municipios_brasileiros 
		WHERE (normalize_text(nome_cidade) LIKE ? OR normalize_text(nome_cidade) LIKE ?)
		AND (? = '' OR uf = ?)
		ORDER BY similarity_rank, nome_cidade
		LIMIT ?
	`,
		nomeNormalizado,         // exact match
		nomeNormalizado+"%",     // starts with
		"%"+nomeNormalizado+"%", // contains
		nomeNormalizado+"%",     // where clause starts with
		"%"+nomeNormalizado+"%", // where clause contains
		ufNormalizado,           // uf filter check
		ufNormalizado,           // uf filter value
		limite,
	)

	err := query.Find(&municipios).Error
	return municipios, err
}

// ContarMunicipios retorna o total de municípios cadastrados
func (s *MunicipioService) ContarMunicipios() (int64, error) {
	var count int64
	err := s.db.Model(&models.MunicipioBrasileiro{}).Count(&count).Error
	return count, err
}

// normalizeText remove acentos e converte para minúsculas
func (s *MunicipioService) normalizeText(text string) string {
	// Remove espaços extras
	text = strings.TrimSpace(text)

	// Converte para minúsculas
	text = strings.ToLower(text)

	// Remove acentos
	replacements := map[string]string{
		"á": "a", "à": "a", "â": "a", "ã": "a", "ä": "a", "å": "a",
		"é": "e", "è": "e", "ê": "e", "ë": "e",
		"í": "i", "ì": "i", "î": "i", "ï": "i",
		"ó": "o", "ò": "o", "ô": "o", "õ": "o", "ö": "o",
		"ú": "u", "ù": "u", "û": "u", "ü": "u",
		"ý": "y", "ÿ": "y",
		"ñ": "n", "ç": "c",
	}

	for accented, plain := range replacements {
		text = strings.ReplaceAll(text, accented, plain)
	}

	// Remove caracteres especiais exceto espaços e hífens
	reg := regexp.MustCompile(`[^a-z0-9\s\-]`)
	text = reg.ReplaceAllString(text, "")

	return text
}
