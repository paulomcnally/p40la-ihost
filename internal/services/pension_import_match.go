package services

import (
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// MatchResult describe la categoría sugerida para un concepto.
type MatchResult struct {
	CategoryID *int64
	Confidence float64
	Reason     string
	Keyword    string
	Mixed      bool
}

// matchCategory aplica nombre normalizado, palabras clave y fuzzy (SPEC-119).
func matchCategory(concept string, categories []models.PensionCategory, keywords []models.PensionCategoryKeyword) MatchResult {
	norm := normalizeText(concept)
	if norm == "" {
		return MatchResult{}
	}

	// 1) Nombre exacto.
	for _, c := range categories {
		if normalizeText(c.Name) == norm {
			id := c.ID
			return MatchResult{CategoryID: &id, Confidence: 1, Reason: "name"}
		}
	}

	// 2) Palabras clave contenidas como frase.
	keywordHits := map[int64]*models.PensionCategoryKeyword{}
	for i := range keywords {
		k := &keywords[i]
		if k.NormalizedKeyword == "" {
			continue
		}
		if containsPhrase(norm, k.NormalizedKeyword) {
			prev, ok := keywordHits[k.PensionCategoryID]
			if !ok || len(k.NormalizedKeyword) > len(prev.NormalizedKeyword) {
				keywordHits[k.PensionCategoryID] = k
			}
		}
	}
	if len(keywordHits) > 1 {
		return MatchResult{Mixed: true}
	}
	for id, hit := range keywordHits {
		categoryID := id
		return MatchResult{CategoryID: &categoryID, Confidence: 0.95, Reason: "keyword", Keyword: hit.Keyword}
	}

	// 3) Fuzzy contra el nombre de la categoría.
	bestScore := 0.0
	var bestID int64
	for _, c := range categories {
		score := similarity(norm, normalizeText(c.Name))
		if score > bestScore {
			bestScore = score
			bestID = c.ID
		}
	}
	if bestScore >= 0.85 {
		id := bestID
		return MatchResult{CategoryID: &id, Confidence: 0.8, Reason: "fuzzy"}
	}

	return MatchResult{}
}

func containsPhrase(text, phrase string) bool {
	return strings.Contains(" "+text+" ", " "+phrase+" ")
}

// similarity devuelve 1 - distancia/maxlen (0 a 1).
func similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1
	}
	return 1 - float64(levenshtein(a, b))/float64(maxLen)
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
