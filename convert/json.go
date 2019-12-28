package convert

import (
	"database/sql"
	"github.com/ghjan/xlsxtodb/pkg/utils"
	"github.com/tealeg/xlsx"
)

func ConvertFromJson(c *Columns, db *sql.DB, driverName, tableName string, titles []StringValue) (err error) {
	for _, item := range titles{
		c.sourceColumns = append(c.sourceColumns, item)
	}

	c.ParaseColumns()

}

func ConvertJsonToDB(db *sql.DB, driverName, tableName, jsonFileName string, titles []StringValue) {
	jsonFile, err := xlsx.OpenFile(jsonFileName)
	if err != nil {
		utils.Checkerr(err, jsonFileName)
	}

	c := new(Columns)
	c.tableColumnMap, err = GetTableColumns(db, driverName, tableName)
	utils.Checkerr(err, "")
	err = ConvertFromJson(c, db, driverName, tableName, titles)
}
