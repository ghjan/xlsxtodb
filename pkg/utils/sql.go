package utils

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func FetchRow(db *sql.DB, sqlstr string, args ...interface{}) (*map[string]string, error) {
	stmtOut, err := db.Prepare(sqlstr)
	Checkerr(err, sqlstr)
	defer stmtOut.Close()

	rows, err := stmtOut.Query(args...)
	Checkerr(err, sqlstr)
	defer rows.Close()

	columns, err := rows.Columns()
	Checkerr(err, sqlstr)

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	ret := make(map[string]string, len(scanArgs))

	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		Checkerr(err, "")
		var value string

		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			ret[columns[i]] = value
		}
		break //get the first row only
	}
	return &ret, nil
}
