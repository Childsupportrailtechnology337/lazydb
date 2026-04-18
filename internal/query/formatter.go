package query

import "strings"

// Format applies basic SQL formatting to a query string.
func Format(query string) string {
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "LEFT JOIN", "RIGHT JOIN",
		"INNER JOIN", "OUTER JOIN", "ON", "AND", "OR", "ORDER BY",
		"GROUP BY", "HAVING", "LIMIT", "OFFSET", "INSERT INTO",
		"VALUES", "UPDATE", "SET", "DELETE FROM", "CREATE TABLE",
	}

	result := query
	for _, kw := range keywords {
		lower := strings.ToLower(kw)
		result = strings.ReplaceAll(result, lower, kw)
	}

	return result
}
