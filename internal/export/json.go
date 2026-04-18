package export

import (
	"encoding/json"
	"io"

	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// JSON writes query results as JSON.
func JSON(w io.Writer, result *db.QueryResult) error {
	var rows []map[string]string
	for _, row := range result.Rows {
		m := make(map[string]string)
		for i, col := range result.Columns {
			if i < len(row) {
				m[col] = row[i]
			}
		}
		rows = append(rows, m)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rows)
}
