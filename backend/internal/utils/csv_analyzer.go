package utils

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// CSVAnalysisResult contém informações sobre a análise do CSV
type CSVAnalysisResult struct {
	Delimiter           rune    `json:"delimiter"`         // ',' ou ';'
	NumericSeparator    string  `json:"numeric_separator"` // '.' ou ','
	HasHeader           bool    `json:"has_header"`
	Columns             int     `json:"columns"`
	SampleRows          int     `json:"sample_rows"`
	DelimiterConfidence float64 `json:"delimiter_confidence"` // 0.0 a 1.0
}

// AnalyzeCSV analisa um arquivo CSV para detectar delimitador e formato numérico
func AnalyzeCSV(reader io.Reader) (*CSVAnalysisResult, error) {
	// Ler as primeiras linhas para análise
	scanner := bufio.NewScanner(reader)
	var lines []string
	maxLines := 10 // Analisar até 10 linhas

	for i := 0; i < maxLines && scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("arquivo vazio")
	}

	// Detectar delimitador
	delimiter, confidence := detectDelimiter(lines)

	// Detectar formato numérico baseado no delimitador
	numericSeparator := "."
	if delimiter == ';' {
		numericSeparator = ","
	}

	// Contar colunas usando o delimitador detectado
	columns := countColumns(lines[0], delimiter)

	result := &CSVAnalysisResult{
		Delimiter:           delimiter,
		NumericSeparator:    numericSeparator,
		HasHeader:           hasHeader(lines, delimiter),
		Columns:             columns,
		SampleRows:          len(lines),
		DelimiterConfidence: confidence,
	}

	return result, nil
}

// detectDelimiter detecta o delimitador mais provável analisando as linhas
func detectDelimiter(lines []string) (rune, float64) {
	if len(lines) == 0 {
		return ',', 0.0
	}

	delimiters := []rune{',', ';'}
	scores := make(map[rune]float64)

	for _, delimiter := range delimiters {
		score := analyzeDelimiterConsistency(lines, delimiter)
		scores[delimiter] = score
	}

	// Escolher delimitador com maior score
	bestDelimiter := ','
	bestScore := scores[',']

	if scores[';'] > bestScore {
		bestDelimiter = ';'
		bestScore = scores[';']
	}

	return bestDelimiter, bestScore
}

// analyzeDelimiterConsistency analisa a consistência de um delimitador
func analyzeDelimiterConsistency(lines []string, delimiter rune) float64 {
	if len(lines) < 2 {
		return 0.0
	}

	delimiterStr := string(delimiter)

	// Contar colunas na primeira linha
	firstLineColumns := len(strings.Split(lines[0], delimiterStr))

	if firstLineColumns < 2 {
		return 0.0 // Delimitador deve criar pelo menos 2 colunas
	}

	consistentLines := 0
	totalLines := len(lines)

	for _, line := range lines {
		columns := len(strings.Split(line, delimiterStr))
		// Aceitar variação de ±1 coluna (para lidar com campos vazios)
		if columns >= firstLineColumns-1 && columns <= firstLineColumns+1 {
			consistentLines++
		}
	}

	consistency := float64(consistentLines) / float64(totalLines)

	// Bonus para delimitadores que criam mais colunas (mais estruturado)
	columnBonus := float64(firstLineColumns) * 0.1
	if columnBonus > 0.3 {
		columnBonus = 0.3
	}

	return consistency + columnBonus
}

// countColumns conta o número de colunas na primeira linha
func countColumns(line string, delimiter rune) int {
	delimiterStr := string(delimiter)
	return len(strings.Split(line, delimiterStr))
}

// hasHeader tenta detectar se a primeira linha é um cabeçalho
func hasHeader(lines []string, delimiter rune) bool {
	if len(lines) < 2 {
		return false
	}

	delimiterStr := string(delimiter)
	firstLine := strings.Split(lines[0], delimiterStr)

	// Heurística: se a primeira linha contém principalmente texto não-numérico
	// e palavras típicas de cabeçalho, provavelmente é header
	headerWords := []string{
		"name", "nome", "price", "preço", "preco", "description", "descricao",
		"sku", "code", "codigo", "category", "categoria", "brand", "marca",
		"stock", "estoque", "quantity", "quantidade", "barcode", "codigo_barra",
	}

	headerCount := 0
	numericPattern := regexp.MustCompile(`^\d+([.,]\d+)*$`)

	for _, field := range firstLine {
		field = strings.ToLower(strings.TrimSpace(field))
		field = strings.Trim(field, `"'`)

		// Verificar se contém palavras de cabeçalho
		for _, headerWord := range headerWords {
			if strings.Contains(field, headerWord) {
				headerCount++
				break
			}
		}

		// Se o campo for numérico, diminui a probabilidade de ser header
		if numericPattern.MatchString(field) {
			headerCount--
		}
	}

	// Se mais de 30% dos campos parecem ser header, considerar como header
	return float64(headerCount)/float64(len(firstLine)) > 0.3
}

// NormalizeNumericValue normaliza valores numéricos baseado no formato detectado
func NormalizeNumericValue(value string, numericSeparator string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)

	if value == "" {
		return value
	}

	// Se o separador numérico é vírgula, converter para ponto para processamento padrão
	if numericSeparator == "," {
		// Trocar vírgula por ponto para decimais
		// Exemplo: "10,50" -> "10.50"
		value = strings.ReplaceAll(value, ",", ".")
	}

	return value
}

// ParseCSVWithDetectedDelimiter analisa e parseia CSV com detecção automática
func ParseCSVWithDetectedDelimiter(reader io.Reader) ([][]string, *CSVAnalysisResult, error) {
	// Primeiro, ler todo o conteúdo para análise
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler conteúdo: %w", err)
	}

	// Analisar formato
	contentReader := strings.NewReader(string(content))
	analysis, err := AnalyzeCSV(contentReader)
	if err != nil {
		return nil, nil, fmt.Errorf("erro na análise: %w", err)
	}

	// Parsear CSV com delimitador detectado
	contentReader = strings.NewReader(string(content))
	csvReader := csv.NewReader(contentReader)
	csvReader.Comma = analysis.Delimiter
	csvReader.FieldsPerRecord = -1 // Permitir registros com número variável de campos

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, analysis, fmt.Errorf("erro ao parsear CSV: %w", err)
	}

	return records, analysis, nil
}
