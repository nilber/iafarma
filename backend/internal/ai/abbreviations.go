package ai

import (
	"regexp"
	"strings"
)

// PharmacyAbbreviations contém um mapeamento de palavras completas para suas possíveis abreviações
var PharmacyAbbreviations = map[string][]string{
	// Formas farmacêuticas
	"comprimidos": {"comp", "cpds", "cpr", "comprs"},
	"cápsulas":    {"caps", "cps", "cap"},
	"drágeas":     {"drag", "drg"},
	"tabletes":    {"tab", "tbl"},
	"pastilhas":   {"past", "pst"},
	"sachês":      {"sach", "sch"},
	"ampolas":     {"amp", "ampl"},
	"frascos":     {"fr", "fras", "frasco"},
	"cartuchos":   {"cart", "cartu"},
	"seringas":    {"ser", "sir"},
	"tubos":       {"tub", "tb"},
	"bisnagas":    {"bisn", "bsn"},
	"gotas":       {"gts", "gt"},
	"colírio":     {"col", "colir"},
	"xarope":      {"xar", "xpe"},
	"suspensão":   {"susp", "sus"},
	"solução":     {"sol", "solu"},
	"gel":         {"gel", "gl"},
	"pomada":      {"pom", "pm"},
	"creme":       {"cr", "crem"},
	"loção":       {"loc", "loç"},
	"spray":       {"spr", "sp"},
	"aerossol":    {"aer", "aero"},
	"inalador":    {"inal", "inhal"},
	"nebulização": {"neb", "nebul"},
	"enema":       {"en", "enem"},
	"supositório": {"sup", "supos"},
	"óvulo":       {"óv", "ovul"},
	"adesivo":     {"ades", "ad"},
	"emplastro":   {"empl", "emp"},
	"desodorante": {"des", "desodor"},

	// Unidades de medida
	"miligramas":       {"mg", "mgr"},
	"gramas":           {"g", "gr", "gm"},
	"microgramas":      {"mcg", "μg", "ug"},
	"miliequivalentes": {"meq", "mEq"},
	"unidades":         {"ui", "u", "un", "uns", "unds", "unids"},
	"mililitros":       {"ml", "mL"},
	"litros":           {"l", "L", "lt"},
	"quilograma":       {"kg", "kilo"},

	// Apresentações comuns
	"envelope":  {"env", "envl"},
	"unidade":   {"un", "und", "unid"},
	"embalagem": {"emb", "embal"},
	"caixa":     {"cx", "cxa"},
	"frasco":    {"fr", "frs", "frasc"},

	// Termos farmacêuticos específicos
	"liberação":     {"lib", "lber"},
	"prolongada":    {"prol", "pr"},
	"retardada":     {"ret", "retard"},
	"revestido":     {"rev", "rvst"},
	"mastigável":    {"mast", "mastig"},
	"sublingual":    {"subl", "sl"},
	"injetável":     {"inj", "inject"},
	"tópico":        {"top", "tóp"},
	"oftálmico":     {"oft", "oftal"},
	"nasal":         {"nas", "nsl"},
	"oral":          {"or", "vo"},
	"intramuscular": {"im", "i.m."},
	"intravenoso":   {"iv", "i.v."},
	"subcutâneo":    {"sc", "s.c."},

	// Concentrações e dosagens
	"concentração": {"conc", "concent"},
	"dosagem":      {"dos", "dosag"},
	"forte":        {"ft", "for"},
	"extra":        {"ext", "x"},
	"máximo":       {"max", "máx"},
	"mínimo":       {"min", "mín"},

	// Marcas e fabricantes comuns (podem ser expandidos)
	"genérico":   {"gen", "genér"},
	"similar":    {"sim", "simil"},
	"referência": {"ref", "refer"},
}

// ExpandQueryWithAbbreviations expande a query de busca incluindo abreviações
func ExpandQueryWithAbbreviations(query string) string {
	if strings.TrimSpace(query) == "" {
		return query
	}

	// Normalizar a query
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	// Dividir em palavras
	words := regexp.MustCompile(`\s+`).Split(normalizedQuery, -1)

	var expandedTerms []string

	for _, word := range words {
		// Limpar a palavra de pontuação
		cleanWord := regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		if cleanWord == "" {
			continue
		}

		// Verificar se a palavra tem correspondência nas abreviações
		found := false
		for fullWord, abbreviations := range PharmacyAbbreviations {
			// Se a palavra é uma das formas completas, incluir as abreviações
			if cleanWord == strings.ToLower(fullWord) {
				// Criar termo OR com todas as variações
				orTerms := []string{cleanWord}
				for _, abbrev := range abbreviations {
					orTerms = append(orTerms, strings.ToLower(abbrev))
				}
				expandedTerms = append(expandedTerms, "("+strings.Join(orTerms, " | ")+")")
				found = true
				break
			}

			// Se a palavra é uma abreviação, incluir a forma completa
			for _, abbrev := range abbreviations {
				if cleanWord == strings.ToLower(abbrev) {
					// Criar termo OR com a forma completa e outras abreviações
					orTerms := []string{cleanWord, strings.ToLower(fullWord)}
					for _, otherAbbrev := range abbreviations {
						if strings.ToLower(otherAbbrev) != cleanWord {
							orTerms = append(orTerms, strings.ToLower(otherAbbrev))
						}
					}
					expandedTerms = append(expandedTerms, "("+strings.Join(orTerms, " | ")+")")
					found = true
					break
				}
			}

			if found {
				break
			}
		}

		// Se não encontrou correspondência, manter a palavra original
		if !found {
			expandedTerms = append(expandedTerms, cleanWord)
		}
	}

	// Retornar os termos unidos com AND
	return strings.Join(expandedTerms, " & ")
}

// BuildAdvancedSearchQuery constrói uma query de busca avançada para PostgreSQL FTS
func BuildAdvancedSearchQuery(originalQuery string) string {
	expanded := ExpandQueryWithAbbreviations(originalQuery)

	// Log para debug
	if expanded != strings.ToLower(originalQuery) {
		return expanded
	}

	// Se não houve expansão, usar a lógica original com AND
	words := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(originalQuery), -1)
	var cleanWords []string

	for _, word := range words {
		cleanWord := regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		if cleanWord != "" {
			cleanWords = append(cleanWords, strings.ToLower(cleanWord))
		}
	}

	return strings.Join(cleanWords, " & ")
}

// GetAbbreviationsForWord retorna todas as abreviações para uma palavra específica
func GetAbbreviationsForWord(word string) []string {
	normalizedWord := strings.ToLower(strings.TrimSpace(word))

	// Procurar por palavra completa
	if abbreviations, exists := PharmacyAbbreviations[normalizedWord]; exists {
		return abbreviations
	}

	// Procurar por abreviação
	for fullWord, abbreviations := range PharmacyAbbreviations {
		for _, abbrev := range abbreviations {
			if strings.ToLower(abbrev) == normalizedWord {
				result := []string{fullWord}
				for _, otherAbbrev := range abbreviations {
					if strings.ToLower(otherAbbrev) != normalizedWord {
						result = append(result, otherAbbrev)
					}
				}
				return result
			}
		}
	}

	return nil
}
