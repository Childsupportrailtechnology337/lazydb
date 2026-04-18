package ui

import (
	"sort"
	"strings"
	"unicode"
)

// FuzzyMatch represents a single fuzzy match result.
type FuzzyMatch struct {
	Str        string
	Index      int
	Score      int
	MatchedPos []int
}

// FuzzyFind performs fuzzy matching of a query against a list of strings.
// Returns matches sorted by score (best first).
func FuzzyFind(query string, items []string) []FuzzyMatch {
	if query == "" {
		matches := make([]FuzzyMatch, len(items))
		for i, s := range items {
			matches[i] = FuzzyMatch{Str: s, Index: i, Score: 0}
		}
		return matches
	}

	query = strings.ToLower(query)
	var matches []FuzzyMatch

	for idx, item := range items {
		score, positions := fuzzyScore(query, item)
		if score > 0 {
			matches = append(matches, FuzzyMatch{
				Str:        item,
				Index:      idx,
				Score:      score,
				MatchedPos: positions,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// fuzzyScore calculates a fuzzy match score.
// Returns 0 if no match. Higher score = better match.
func fuzzyScore(query, target string) (int, []int) {
	lower := strings.ToLower(target)
	score := 0
	qi := 0
	var positions []int

	for ti := 0; ti < len(lower) && qi < len(query); ti++ {
		if lower[ti] == query[qi] {
			positions = append(positions, ti)
			qi++

			// Bonus for consecutive matches
			if len(positions) > 1 && positions[len(positions)-1] == positions[len(positions)-2]+1 {
				score += 10
			}

			// Bonus for matching at start
			if ti == 0 {
				score += 15
			}

			// Bonus for matching after separator (_, -, space, camelCase)
			if ti > 0 {
				prev := rune(target[ti-1])
				curr := rune(target[ti])
				if prev == '_' || prev == '-' || prev == ' ' || prev == '.' {
					score += 10
				}
				// camelCase boundary
				if unicode.IsLower(prev) && unicode.IsUpper(curr) {
					score += 8
				}
			}

			score += 5 // Base match score

			// Exact case match bonus
			if target[ti] == query[qi-1] {
				score += 2
			}
		}
	}

	if qi < len(query) {
		return 0, nil // Not all characters matched
	}

	// Bonus for shorter targets (more relevant)
	score += max(0, 50-len(target))

	// Bonus for exact match
	if lower == query {
		score += 100
	}

	// Bonus for prefix match
	if strings.HasPrefix(lower, query) {
		score += 50
	}

	return score, positions
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// HighlightFuzzyMatch renders a string with matched characters highlighted.
func HighlightFuzzyMatch(s string, positions []int, matchStyle, normalStyle func(string) string) string {
	if len(positions) == 0 {
		return normalStyle(s)
	}

	posSet := make(map[int]bool)
	for _, p := range positions {
		posSet[p] = true
	}

	var result strings.Builder
	for i, ch := range s {
		if posSet[i] {
			result.WriteString(matchStyle(string(ch)))
		} else {
			result.WriteString(normalStyle(string(ch)))
		}
	}
	return result.String()
}

// FuzzyFilter is a reusable fuzzy filter component for lists.
type FuzzyFilter struct {
	query   string
	items   []string
	matches []FuzzyMatch
	cursor  int
	active  bool
}

// NewFuzzyFilter creates a new fuzzy filter.
func NewFuzzyFilter() FuzzyFilter {
	return FuzzyFilter{}
}

// SetItems sets the items to filter.
func (f *FuzzyFilter) SetItems(items []string) {
	f.items = items
	f.applyFilter()
}

// SetQuery sets the filter query.
func (f *FuzzyFilter) SetQuery(q string) {
	f.query = q
	f.cursor = 0
	f.applyFilter()
}

// AddChar adds a character to the query.
func (f *FuzzyFilter) AddChar(ch string) {
	f.query += ch
	f.cursor = 0
	f.applyFilter()
}

// Backspace removes the last character.
func (f *FuzzyFilter) Backspace() {
	if len(f.query) > 0 {
		f.query = f.query[:len(f.query)-1]
		f.cursor = 0
		f.applyFilter()
	}
}

// Clear clears the query.
func (f *FuzzyFilter) Clear() {
	f.query = ""
	f.cursor = 0
	f.applyFilter()
}

// Query returns the current query.
func (f *FuzzyFilter) Query() string {
	return f.query
}

// Matches returns the filtered matches.
func (f *FuzzyFilter) Matches() []FuzzyMatch {
	return f.matches
}

// Cursor returns the current cursor position.
func (f *FuzzyFilter) Cursor() int {
	return f.cursor
}

// MoveUp moves the cursor up.
func (f *FuzzyFilter) MoveUp() {
	if f.cursor > 0 {
		f.cursor--
	}
}

// MoveDown moves the cursor down.
func (f *FuzzyFilter) MoveDown() {
	if f.cursor < len(f.matches)-1 {
		f.cursor++
	}
}

// Selected returns the currently selected match, or -1 if none.
func (f *FuzzyFilter) Selected() int {
	if f.cursor < len(f.matches) {
		return f.matches[f.cursor].Index
	}
	return -1
}

// IsActive returns whether the filter is active.
func (f *FuzzyFilter) IsActive() bool {
	return f.active
}

// SetActive sets whether the filter is active.
func (f *FuzzyFilter) SetActive(active bool) {
	f.active = active
	if !active {
		f.query = ""
	}
}

func (f *FuzzyFilter) applyFilter() {
	f.matches = FuzzyFind(f.query, f.items)
}
