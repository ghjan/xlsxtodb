package convert

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ghjan/xlsxtodb/pkg/set"
	"github.com/ghjan/xlsxtodb/pkg/utils"
	"github.com/tealeg/xlsx"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetColumns(db *sql.DB, driverName, tableName string) (*Columns, error) {
	sql := "SELECT * FROM " + utils.EscapeString(driverName, tableName) + " LIMIT 1"
	dbRows, err := db.Query(sql)
	defer dbRows.Close()
	utils.Checkerr(err, sql)
	//获取数据表字段名
	c := new(Columns)
	onlyEnglishColumnMap := map[int]string{}
	if cols, err := dbRows.Columns(); err == nil {
		for index, col := range cols {
			if !utils.HasChineseChar(col) {
				onlyEnglishColumnMap[index+1] = col
			}
		}
	}
	c.tableColumnMap = onlyEnglishColumnMap
	utils.Checkerr(err, sql)
	return c, err
}

func Convert(c *Columns, sheet *xlsx.Sheet, db *sql.DB, dataStartRow int, driverName, tableName string) (err error) {
	err = nil
	if sheet.Rows == nil {
		err = errors.New("该sheet没有数据")
		return
	}
	c.xlsxColumns = sheet.Rows[0].Cells
	c.ParaseColumns()
	//ch := make(chan int, 50)
	rowsNum := len(sheet.Rows) //- dataStartRow + 2
	for rowIndex := dataStartRow - 1; rowIndex < rowsNum; rowIndex++ {
		//ch <- rowIndex
		utils.Checkerr(err, "")
		r := &Row{value: make(map[string]string), sql: "", ot: OtherTable{}}
		tmp := 0
		var fieldNames []string
		var values []string
		needConflictOnFields := ""
		needConflictOnFieldsOther := ""
		updateSetSql := ""
		whereSql := ""
		//var excludedFields []string
		id := ""
		excludedFieldSet := set.New()
		updatedFieldSet := set.New()
		var uniqTogetherMap = map[string]string{}
		for key, value := range c.useColumns {
			if value == nil || len(sheet.Rows) <= 0 {
				fmt.Printf("c.useColumns:%#v\n", c.useColumns)
				continue
			}
			if key >= len(sheet.Rows[rowIndex].Cells) || sheet.Rows[rowIndex].Cells[key] == nil {
				r.value[value[0]] = ""
			} else {
				r.value[value[0]] = sheet.Rows[rowIndex].Cells[key].String()
			}
			//解析内容
			if value[0] == ":other" {
				r.ot.value = strings.Split(r.value[value[0]], "|")
				sqlOther := "SELECT * FROM " + utils.EscapeString(driverName, r.ot.value[0])
				rows, err := db.Query(sqlOther)
				utils.Checkerr(err, sqlOther)
				r.ot.columns, err = rows.Columns()
				rows.Close()
				utils.Checkerr(err, sqlOther)

				for _, v := range value {
					if strings.Index(v, "unique=") >= 0 {
						needConflictOnFieldsOther = strings.Join(strings.Split(strings.TrimPrefix(v,
							"unique="), "+"), ",")
					}
				}
			} else {
				if len(value) > 1 {
					switch value[1] {
					case "unique":
						if id != "" {
							continue
						}
						uniqueSql := "SELECT id, " + value[0] + " FROM  " +
							utils.EscapeString(driverName, tableName) + "  WHERE  " + value[0] + "  = '" +
							utils.EscapeSpecificChar(r.value[value[0]]) + "'"
						//fmt.Printf("uniqueSql:%s\n", uniqueSql)
						result, _ := utils.FetchRow(db, uniqueSql)
						id = (*result)["id"]
						if id != "" {
							if needConflictOnFields == "" {
								needConflictOnFields = value[0]
							} else {
								needConflictOnFields += "," + value[0]
							}
						}
					case "uniq_together":
						uniqTogetherMap[value[0]] = utils.EscapeSpecificChar(r.value[value[0]])
						if needConflictOnFields == "" {
							needConflictOnFields = value[0]
						} else {
							needConflictOnFields += "," + value[0]
						}
					case "password":
						tmpvalue := strings.Split(r.value[value[0]], "|")
						if len(tmpvalue) == 2 {
							if []byte(tmpvalue[1])[0] == ':' {
								if _, ok := r.value[string([]byte(tmpvalue[1])[1:])]; ok {
									r.value[value[0]] = tmpvalue[0] + r.value[string([]byte(tmpvalue[1])[1:])]
								} else {
									msg := "[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) +
										"]密码盐" + string([]byte(tmpvalue[1])[1:]) + "字段不存在，自动跳过"
									//fmt.Println(msg)
									err = errors.New(msg)
									//sign <- "error"
									return
								}
							} else {
								r.value[value[0]] += tmpvalue[1]
							}
						} else {
							r.value[value[0]] = tmpvalue[0]
						}
						switch value[2] {
						case "md5":
							r.value[value[0]] = string(md5.New().Sum([]byte(r.value[value[0]])))
						case "bcrypt":
							pass, _ := bcrypt.GenerateFromPassword([]byte(r.value[value[0]]), 13)
							r.value[value[0]] = string(pass)
						}
					case "find":
						result, _ := utils.FetchRow(db, "SELECT  "+value[3]+"  FROM  "+utils.EscapeString(driverName, value[2])+"  WHERE "+value[4]+" = '"+r.value[value[0]]+"'")
						if (*result)["id"] == "" {
							//sign <- "error"
							msg := "[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]表 " +
								value[2] + " 中没有找到 " + value[4] + " 为 " + r.value[value[0]] + " 的数据，自动跳过"
							//fmt.Println(msg)
							err = errors.New(msg)
							return
						}
						excludedFieldSet.Add(value[0])
						updatedFieldSet.Add(value[0])
						r.value[value[0]] = (*result)["id"]
					}
				}
				var pro bool
				r.value[value[0]], pro = ParseValue(r.value[value[0]])

				if r.value[value[0]] != "" {
					fieldNames = append(fieldNames, value[0])
					values = append(values, utils.EscapeValuesString(driverName, r.value[value[0]]))
					updatedFieldSet.Add(value[0])
					if !pro {
						excludedFieldSet.Add(value[0])
					}
					tmp++
				}
			}
		}
		insertSql, updateSetSql, whereSql := GetUpdateSql(driverName, tableName, fieldNames, values,
			needConflictOnFields, updatedFieldSet, excludedFieldSet)
		r.sql = insertSql
		if needConflictOnFields != "" && updateSetSql != "" && whereSql != "" {
			r.sql += " ON CONFLICT (" + needConflictOnFields + ") DO UPDATE SET " +
				updateSetSql + whereSql
		}
		r.sql += " RETURNING id"
		db.QueryRow(r.sql + ";").Scan(&r.insertID)

		idOfMainRecord := int(r.insertID)
		if idOfMainRecord == 0 {
			fmt.Println(r.sql)
			if id != "" {
				idOfMainRecord, _ = strconv.Atoi(id)
			} else if len(uniqTogetherMap) > 0 {
				uniqTogetherSql := " select id from " + tableName + " where "
				indexTemp := 0
				for field, value := range uniqTogetherMap {
					if indexTemp > 0 {
						uniqTogetherSql += " and "
					}
					uniqTogetherSql += field + "=" + utils.EscapeValuesString(driverName, value)
					indexTemp += 1
				}
				result, _ := utils.FetchRow(db, uniqTogetherSql)
				id := (*result)["id"]
				if id != "" {
					idOfMainRecord, _ = strconv.Atoi(id)
				}
			}
		}
		if idOfMainRecord == 0 {
			err = errors.New("id is 0")
			return
		}
		utils.Checkerr(err, r.sql)
		if idOfMainRecord > 0 && err == nil {
			//更新id自增长值
			idSeqField := tableName + "_id_seq"
			// SELECT setval('', (SELECT max(id) FROM tableName))
			sqlIdSeq := "SELECT setval('" + idSeqField + "', (SELECT max(id) FROM " + tableName + "))"
			db.Query(sqlIdSeq)
		}

		//执行附表操作
		updateSetSql = ""
		whereSql = ""
		excludedFieldSet = set.New()
		updatedFieldSet = set.New()
		tableNameOther := ""
		if r.ot.value != nil {
			tableNameOther = r.ot.value[0]
			tmp = 0
			var fieldNames []string
			var values []string
			for key, value := range r.ot.columns {
				var pro bool
				r.ot.value[key+1], pro = ParseValue(r.ot.value[key+1])
				if r.ot.value[key+1] == ":id" {
					r.ot.value[key+1] = strconv.Itoa(idOfMainRecord)
				}
				if r.ot.value[key+1] != "" {
					if value != "" {
						fieldNames = append(fieldNames, value)
						values = append(values, r.ot.value[key+1])
						updatedFieldSet.Add(value)
						if !pro {
							excludedFieldSet.Add(value)
						}
						tmp++
					}
				}
			}
			insertSql, updateSetSql, whereSql = GetUpdateSql(driverName, tableNameOther, fieldNames, values,
				needConflictOnFieldsOther, updatedFieldSet, excludedFieldSet)
			r.ot.sql = insertSql
			if needConflictOnFieldsOther != "" {
				r.ot.sql += " ON CONFLICT (" + needConflictOnFieldsOther + ") DO UPDATE SET " +
					updateSetSql + whereSql
			}

			db.QueryRow(r.ot.sql + ";")
		}
		fmt.Println("[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]导入数据成功")
		//sign <- "success"
		//}//end of go func
	}
	//for rowIndex := dataStartRow - 1; rowIndex <= rowsNum; rowIndex++ {
	//	<-sign
	//}
	return
}

func GetUpdateSql(driverName, tableName string, fieldNames []string, values []string, needConflictOnFields string,
	updatedFieldSet *set.StringSet, excludedFieldSet *set.StringSet) (sql string, updateSetSql string, whereSql string) {
	sql = "INSERT INTO  " + utils.EscapeString(driverName, tableName)
	sql += " (" + strings.Join(fieldNames, ",") + ")"
	sql += " VALUES (" + strings.Join(values, ",") + ")"
	conflictFieldSet := set.NewFromSlice(strings.Split(needConflictOnFields, ","))
	updatedFieldSet = updatedFieldSet.Difference(conflictFieldSet).Difference(NeedNotUpdateFieldSet)
	excludedFieldSet = excludedFieldSet.Difference(conflictFieldSet)
	if needConflictOnFields != "" && excludedFieldSet.Len() > 0 {
		for idx, fieldName := range updatedFieldSet.List() {
			if idx > 0 {
				updateSetSql += ","
			} else {
				updateSetSql += " "
			}
			updateSetSql += fieldName + "=excluded." + fieldName
		}
		for idx, fieldName := range excludedFieldSet.List() {
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
	for key, value := range c.xlsxColumns {
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
				if columnval[0] == value {
					c.useColumns[key] = columnval
				}
			}
		}
	}
	if len(c.useColumns) < 1 {
		fmt.Println("数据表与xlsx表格不对应")
		os.Exit(-1)
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
		result = strings.Replace(Substr(base64.StdEncoding.EncodeToString(utils.Krand(32, utils.KC_RAND_KIND_ALL)), 0, 32), "+/", "_-", -1)
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

func ConverExcelToDB(db *sql.DB, driverName, tableName, excelFileName, sheets string, dataStartRow int) {
	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		utils.Checkerr(err, excelFileName)
	}
	sheetSlice := strings.Split(sheets, ",")
	sign := make(chan string, len(sheetSlice))
	for _, sheetIndex := range sheetSlice {
		go func(sheetIndex string) {
			c, _ := GetColumns(db, driverName, tableName)
			nIndex, err := strconv.Atoi(sheetIndex)
			utils.Checkerr(err, "")
			if nIndex >= len(xlFile.Sheets) {
				fmt.Printf("Sheet index should small then length of sheets.sheetIndex:%d, "+
					"len(xlFile.Sheets):%d\n", sheetIndex, len(xlFile.Sheets))
				return
			}
			sheet_ := xlFile.Sheets[nIndex]
			fmt.Printf("--------sheetIndex%s, %s----------\n", sheetIndex, sheet_.Name)
			err = Convert(c, sheet_, db, dataStartRow, driverName, tableName)
			if utils.Checkerr2(err, fmt.Sprintf("sheetIndex:%d", nIndex)) {
				sign <- "error"
			} else {
				sign <- "success"
			}
		}(sheetIndex)
	}
	for sheetIndex := dataStartRow - 1; sheetIndex <= len(sheetSlice); sheetIndex++ {
		<-sign
	}
}
