package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// SQL writes query results as SQL INSERT statements.
func SQL(w io.Writer, tableName string, result *db.QueryResult) error {
	for _, row := range result.Rows {
		var values []string
		for _, cell := range row {
			if cell == "<NULL>" {
				values = append(values, "NULL")
			} else {
				escaped := strings.ReplaceAll(cell, "'", "''")
				values = append(values, "'"+escaped+"'")
			}
		}
		stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n",
			tableName,
			strings.Join(result.Columns, ", "),
			strings.Join(values, ", "),
		)
		if _, err := io.WriteString(w, stmt); err != nil {
			return err
		}
	}
	return nil
}
