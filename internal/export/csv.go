package export

import (
	"encoding/csv"
	"io"

	"github.com/aymenhmaidiwastaken/lazydb/internal/db"
)

// CSV writes query results as CSV.
func CSV(w io.Writer, result *db.QueryResult) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(result.Columns); err != nil {
		return err
	}

	for _, row := range result.Rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
