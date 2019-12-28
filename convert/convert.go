package convert

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/ghjan/xlsxtodb/pkg/set"
	"github.com/ghjan/xlsxtodb/pkg/utils"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetTableColumns(db *sql.DB, driverName, tableName string) (tableColumnMap map[int]string, err error) {
	sql := "SELECT * FROM " + utils.EscapeString(driverName, tableName) + " LIMIT 1"
	dbRows, err := db.Query(sql)
	defer dbRows.Close()
	utils.Checkerr(err, sql)
	//获取数据表字段名
	tableColumnMap = make(map[int]string, 0)
	if cols, err := dbRows.Columns(); err == nil {
		for index, col := range cols {
			if !utils.HasChineseChar(col) {
				tableColumnMap[index+1] = col
			}
		}
	}
	//tableColumnMap = onlyEnglishColumnMap
	utils.Checkerr(err, sql)
	return
}

func GetUpdateSql(driverName, tableName string, fieldNames []string, values []string, needConflictOnFields string,
	updatedFieldSet *set.StringSet, distinctExcludedFieldSet *set.StringSet) (sql string,
	updateSetSql string, whereSql string) {
	sql = "INSERT INTO  " + utils.EscapeString(driverName, tableName)
	sql += " (" + strings.Join(fieldNames, ",") + ")"
	sql += " VALUES (" + strings.Join(values, ",") + ")"
	conflictFieldSet := set.NewFromSlice(strings.Split(needConflictOnFields, ","))
	updatedFieldSet = updatedFieldSet.Difference(conflictFieldSet).Difference(NeedNotUpdateFieldSet)
	distinctExcludedFieldSet = distinctExcludedFieldSet.Difference(conflictFieldSet)
	if needConflictOnFields != "" && distinctExcludedFieldSet.Len() > 0 {
		for idx, fieldName := range updatedFieldSet.List() {
			if idx > 0 {
				updateSetSql += ","
			} else {
				updateSetSql += " "
			}
			updateSetSql += fieldName + "=excluded." + fieldName
		}
		for idx, fieldName := range distinctExcludedFieldSet.List() {
			//where	tbl.c3 is distinct from excluded.c3 or tbl.c4 is distinct from excluded.c4;
			if idx > 0 {
				whereSql += " or "
			} else {
				whereSql += " where "
			}
			whereSql += utils.EscapeString(driverName, tableName) + "." + fieldName +
				" is distinct from" + " excluded." + fieldName
		}
	}
	return sql, updateSetSql, whereSql
}

//解析Excel及数据库字段
func (c *Columns) ParaseColumns() {
	c.useColumns = make(map[int][]string)
	hitTableColumnSet := set.New()
	allTableColmnSet := set.New()
	for key, value := range c.sourceColumns {
		thiscolumn := value.String()
		if thiscolumn == "" || utils.HasChineseChar(thiscolumn) {
			// 忽略中文字符的字段名
			continue
		}
		columnval := strings.Split(thiscolumn, "|")
		if columnval == nil || len(columnval) == 0 {
			continue
		}
		if columnval[0] == ":other" {
			c.useColumns[key] = columnval
		} else {
			for _, value := range c.tableColumnMap {
				allTableColmnSet.Add(value)
				if columnval[0] == value {
					c.useColumns[key] = columnval
					hitTableColumnSet.Add(value)
				}
			}
		}
	}

	if len(c.useColumns) < 1 {
		fmt.Println("数据表与xlsx表格不对应")
		os.Exit(-1)
	}
	missTableColumnSet := allTableColmnSet.Difference(hitTableColumnSet)
	generateFieldSet := missTableColumnSet.Intersect(NeedGenerateFieldSet)
	keyNext := len(c.useColumns)
	for _, v := range generateFieldSet.List() {
		c.useColumns[keyNext] = []string{v, "generate"}
		keyNext += 1
	}
}

//解析内容
func ParseValue(val string) (result string, processed bool) {
	switch val {
	case ":null":
		result = ""
		processed = true
	case ":time":
		result = strconv.Itoa(int(time.Now().Unix()))
		processed = true
	case ":random":
		result = strings.Replace(Substr(base64.StdEncoding.EncodeToString(utils.Krand(32,
			utils.KC_RAND_KIND_ALL)), 0, 32), "+/", "_-", -1)
		processed = true
	default:
		result = val
		processed = false
	}
	return result, processed
}

//按长度截取字符串
func Substr(str string, start int, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}

