package ui

import (
	"testing"
)

func TestFuzzyFindExactMatch(t *testing.T) {
	items := []string{"users", "orders", "products", "categories"}
	matches := FuzzyFind("users", items)

	if len(matches) == 0 {
		t.Fatal("expected at least 1 match")
	}
	if matches[0].Str != "users" {
		t.Errorf("best match = %q, want users", matches[0].Str)
	}
}

func TestFuzzyFindPrefixMatch(t *testing.T) {
	items := []string{"user_accounts", "order_details", "user_sessions"}
	matches := FuzzyFind("user", items)

	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	// Both user_ items should match
	matched := map[string]bool{}
	for _, m := range matches {
		matched[m.Str] = true
	}
	if !matched["user_accounts"] || !matched["user_sessions"] {
		t.Errorf("expected both user tables to match, got %v", matched)
	}
}

func TestFuzzyFindSubsequence(t *testing.T) {
	items := []string{"user_accounts", "unique_auth", "umbrella"}
	matches := FuzzyFind("ua", items)

	if len(matches) == 0 {
		t.Fatal("expected matches for subsequence 'ua'")
	}
	// user_accounts should match (u...a)
	found := false
	for _, m := range matches {
		if m.Str == "user_accounts" {
			found = true
		}
	}
	if !found {
		t.Error("user_accounts should match 'ua'")
	}
}

func TestFuzzyFindNoMatch(t *testing.T) {
	items := []string{"users", "orders", "products"}
	matches := FuzzyFind("xyz", items)

	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestFuzzyFindEmpty(t *testing.T) {
	items := []string{"a", "b", "c"}
	matches := FuzzyFind("", items)

	if len(matches) != 3 {
		t.Errorf("empty query should return all items, got %d", len(matches))
	}
}

func TestFuzzyFindCaseInsensitive(t *testing.T) {
	items := []string{"UserAccounts", "ORDER_ITEMS"}
	matches := FuzzyFind("order", items)

	if len(matches) == 0 {
		t.Fatal("case-insensitive search should find ORDER_ITEMS")
	}
	if matches[0].Str != "ORDER_ITEMS" {
		t.Errorf("best match = %q, want ORDER_ITEMS", matches[0].Str)
	}
}

func TestFuzzyFindScoring(t *testing.T) {
	items := []string{"user_metadata", "users", "super_user_cache"}

	matches := FuzzyFind("users", items)
	if len(matches) == 0 {
		t.Fatal("expected matches")
	}
	// Exact match "users" should be first
	if matches[0].Str != "users" {
		t.Errorf("exact match should rank first, got %q", matches[0].Str)
	}
}

func TestFuzzyFilter(t *testing.T) {
	f := NewFuzzyFilter()
	f.SetItems([]string{"users", "orders", "products", "categories", "reviews"})

	f.AddChar("o")
	f.AddChar("r")

	matches := f.Matches()
	if len(matches) == 0 {
		t.Fatal("expected matches for 'or'")
	}

	// orders should be in results
	found := false
	for _, m := range matches {
		if m.Str == "orders" {
			found = true
		}
	}
	if !found {
		t.Error("orders should match 'or'")
	}

	f.Backspace()
	if f.Query() != "o" {
		t.Errorf("query after backspace = %q, want o", f.Query())
	}

	f.Clear()
	if f.Query() != "" {
		t.Errorf("query after clear = %q, want empty", f.Query())
	}
	if len(f.Matches()) != 5 {
		t.Errorf("all items should show when cleared, got %d", len(f.Matches()))
	}
}

func TestHighlightFuzzyMatch(t *testing.T) {
	result := HighlightFuzzyMatch("users", []int{0, 1},
		func(s string) string { return "[" + s + "]" },
		func(s string) string { return s },
	)
	if result != "[u][s]ers" {
		t.Errorf("highlight = %q, want [u][s]ers", result)
	}
}
