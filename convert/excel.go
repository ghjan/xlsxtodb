package convert

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ghjan/xlsxtodb/pkg/set"
	"github.com/ghjan/xlsxtodb/pkg/utils"
	"github.com/google/uuid"
	"github.com/lithammer/shortuuid"
	"github.com/tealeg/xlsx"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
)

func ConvertFromExcel(c *Columns, sheet *xlsx.Sheet, db *sql.DB, dataStartRow int, driverName,
	tableName string) (err error) {
	err = nil
	if sheet.Rows == nil {
		err = errors.New("该sheet没有数据")
		return
	}
	for _, cell := range sheet.Rows[0].Cells {
		c.sourceColumns = append(c.sourceColumns, cell)
	}

	c.ParaseColumns()
	//ch := make(chan int, 50)
	rowsNum := len(sheet.Rows) //- dataStartRow + 2
	for rowIndex := dataStartRow - 1; rowIndex < rowsNum; rowIndex++ {
		//ch <- rowIndex
		utils.Checkerr(err, "")
		r := &Row{value: make(map[string]string), sql: "", ot: OtherTable{}}
		tmp := 0
		// 字段
		var insertIntoFieldNames []string     //INSERT INTO ()
		needConflictOnFields := ""            //CONFLICT ON()
		distinctExcludedFieldSet := set.New() //where tbl.c3 is distinct from excluded.c3 or tbl.c4 is distinct from excluded.c4
		updatedFieldSet := set.New()          // DO UPDATE SET

		var values []string

		needConflictOnFieldsOther := ""
		updateSetSql := ""
		whereSql := ""
		//var excludedFields []string
		id := ""
		returningFields := []string{"id"}
		var rows *sql.Rows
		var uniqTogetherMap = map[string]string{}
		for key, columnFieldValues := range c.useColumns {
			if columnFieldValues == nil || len(sheet.Rows) <= 0 {
				fmt.Printf("c.useColumns:%#v\n", c.useColumns)
				continue
			}
			if key >= len(sheet.Rows[rowIndex].Cells) || sheet.Rows[rowIndex].Cells[key] == nil {
				r.value[columnFieldValues[0]] = ""
			} else {
				r.value[columnFieldValues[0]] = sheet.Rows[rowIndex].Cells[key].String()
			}
			//解析内容
			if columnFieldValues[0] == ":other" {
				r.ot.value = strings.Split(r.value[columnFieldValues[0]], "|")
				sqlOther := "SELECT * FROM " + utils.EscapeString(driverName, r.ot.value[0])
				rows, err = db.Query(sqlOther)
				utils.Checkerr(err, sqlOther)
				r.ot.columns, err = rows.Columns()
				rows.Close()
				utils.Checkerr(err, sqlOther)

				for _, v := range columnFieldValues {
					if strings.Index(v, "unique=") >= 0 {
						needConflictOnFieldsOther = strings.Join(strings.Split(strings.TrimPrefix(v,
							"unique="), "+"), ",")
					}
				}
			} else {
				if len(columnFieldValues) > 1 {
					switch columnFieldValues[1] {
					case "generate":
						if columnFieldValues[0] == "uuid" {
							r.value[columnFieldValues[0]] = uuid.New().String()
							returningFields = append(returningFields, columnFieldValues[0])
						} else if columnFieldValues[0] == "short_uuid" {
							r.value[columnFieldValues[0]] = shortuuid.New()
							returningFields = append(returningFields, columnFieldValues[0])
						} else {
							msg := fmt.Sprintf("columnFieldValues:%#v", columnFieldValues)
							//fmt.Println(msg)
							err = errors.New(msg)
							//sign <- "error"
							return
						}
					case "unique":
						if id != "" {
							continue
						}
						uniqueSql := "SELECT id, " + columnFieldValues[0] + " FROM  " +
							utils.EscapeString(driverName, tableName) + "  WHERE  " + columnFieldValues[0] + "  = '" +
							utils.EscapeSpecificChar(r.value[columnFieldValues[0]]) + "'"
						//fmt.Printf("uniqueSql:%s\n", uniqueSql)
						result, _ := utils.FetchRow(db, uniqueSql)
						id = (*result)["id"]
						if id != "" {
							if needConflictOnFields == "" {
								needConflictOnFields = columnFieldValues[0]
							} else {
								needConflictOnFields += "," + columnFieldValues[0]
							}
						}
					case "uniq_together":
						uniqTogetherMap[columnFieldValues[0]] = utils.EscapeSpecificChar(r.value[columnFieldValues[0]])
						if needConflictOnFields == "" {
							needConflictOnFields = columnFieldValues[0]
						} else {
							needConflictOnFields += "," + columnFieldValues[0]
						}
					case "password":
						tmpvalue := strings.Split(r.value[columnFieldValues[0]], "|")
						if len(tmpvalue) == 2 {
							if []byte(tmpvalue[1])[0] == ':' {
								if _, ok := r.value[string([]byte(tmpvalue[1])[1:])]; ok {
									r.value[columnFieldValues[0]] = tmpvalue[0] +
										r.value[string([]byte(tmpvalue[1])[1:])]
								} else {
									msg := "[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) +
										"]密码盐" + string([]byte(tmpvalue[1])[1:]) + "字段不存在，自动跳过"
									//fmt.Println(msg)
									err = errors.New(msg)
									//sign <- "error"
									return
								}
							} else {
								r.value[columnFieldValues[0]] += tmpvalue[1]
							}
						} else {
							r.value[columnFieldValues[0]] = tmpvalue[0]
						}
						switch columnFieldValues[2] {
						case "md5":
							r.value[columnFieldValues[0]] = string(md5.New().Sum([]byte(r.value[columnFieldValues[0]])))
						case "bcrypt":
							pass, _ := bcrypt.GenerateFromPassword([]byte(r.value[columnFieldValues[0]]), 13)
							r.value[columnFieldValues[0]] = string(pass)
						}
					case "find":
						result, _ := utils.FetchRow(db, "SELECT  "+columnFieldValues[3]+"  FROM  "+
							utils.EscapeString(driverName, columnFieldValues[2])+"  WHERE "+columnFieldValues[4]+
							" = '"+r.value[columnFieldValues[0]]+"'")
						if (*result)["id"] == "" {
							//sign <- "error"
							msg := "[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]表 " +
								columnFieldValues[2] + " 中没有找到 " + columnFieldValues[4] + " 为 " +
								r.value[columnFieldValues[0]] + " 的数据，自动跳过"
							//fmt.Println(msg)
							err = errors.New(msg)
							return
						}
						distinctExcludedFieldSet.Add(columnFieldValues[0])
						updatedFieldSet.Add(columnFieldValues[0])
						r.value[columnFieldValues[0]] = (*result)["id"]
					}
				}
				var pro bool
				r.value[columnFieldValues[0]], pro = ParseValue(r.value[columnFieldValues[0]])

				if r.value[columnFieldValues[0]] != "" {
					insertIntoFieldNames = append(insertIntoFieldNames, columnFieldValues[0])
					values = append(values, utils.EscapeValuesString(driverName, r.value[columnFieldValues[0]]))
					updatedFieldSet.Add(columnFieldValues[0])
					if !pro {
						distinctExcludedFieldSet.Add(columnFieldValues[0])
					}
					tmp++
				}
			}
		}

		if id == "" && len(uniqTogetherMap) > 0 {
			uniqTogetherSql := " select " + strings.Join(returningFields, ",") + " from " + tableName + " where "
			indexTemp := 0
			for field, value := range uniqTogetherMap {
				if indexTemp > 0 {
					uniqTogetherSql += " and "
				}
				uniqTogetherSql += field + "=" + utils.EscapeValuesString(driverName, value)
				indexTemp += 1
			}
			result, _ := utils.FetchRow(db, uniqTogetherSql)
			for _, fieldReturning := range returningFields {
				if fieldReturning == "id" {
					id = (*result)["id"]
				} else { // 如果没有值，就需要update
					if (*result)[fieldReturning] == "" || (*result)[fieldReturning] == "NULL" {
						updatedFieldSet.Add(fieldReturning)
						distinctExcludedFieldSet.Add(fieldReturning)
					} else {
						updatedFieldSet.Remove(fieldReturning)
						distinctExcludedFieldSet.Remove(fieldReturning)
					}
				}
			}
		}
		insertSql, updateSetSql, whereSql := GetUpdateSql(driverName, tableName, insertIntoFieldNames, values,
			needConflictOnFields, updatedFieldSet, distinctExcludedFieldSet)
		r.sql = insertSql
		if needConflictOnFields != "" && updateSetSql != "" && whereSql != "" {
			r.sql += " ON CONFLICT (" + needConflictOnFields + ") DO UPDATE SET " +
				updateSetSql + whereSql
		}
		r.sql += " RETURNING id"
		rows, err = db.Query(r.sql + ";")
		if err == nil {
			rows.Scan(&r.insertID)
		} else {
			fmt.Printf("err:%s\n", err.Error())
			fmt.Printf("r.sql:%s\n", r.sql)
		}
		rows.Close()

		idOfMainRecord := int(r.insertID)
		if idOfMainRecord == 0 {
			fmt.Println(r.sql)
			if id != "" {
				idOfMainRecord, _ = strconv.Atoi(id)
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
			rows, err = db.Query(sqlIdSeq)
			rows.Close()
			utils.Checkerr(err, sqlIdSeq)
		}

		//执行附表操作
		updateSetSql = ""
		whereSql = ""
		distinctExcludedFieldSet = set.New()
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
							distinctExcludedFieldSet.Add(value)
						}
						tmp++
					}
				}
			}
			insertSql, updateSetSql, whereSql = GetUpdateSql(driverName, tableNameOther, fieldNames, values,
				needConflictOnFieldsOther, updatedFieldSet, distinctExcludedFieldSet)
			r.ot.sql = insertSql
			if needConflictOnFieldsOther != "" {
				r.ot.sql += " ON CONFLICT (" + needConflictOnFieldsOther + ") DO UPDATE SET " +
					updateSetSql + whereSql
			}
			rows, err := db.Query(r.ot.sql + ";")
			rows.Close()
			utils.Checkerr(err, r.ot.sql)

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

func ConvertExcelToDB(db *sql.DB, driverName, tableName, excelFileName, sheets string, dataStartRow int) {
	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		utils.Checkerr(err, excelFileName)
	}
	sheetSlice := strings.Split(sheets, ",")
	sign := make(chan string, len(sheetSlice))
	for _, sheetIndex := range sheetSlice {
		go func(sheetIndex string) {
			c := new(Columns)
			c.tableColumnMap, err = GetTableColumns(db, driverName, tableName)
			nIndex, err := strconv.Atoi(sheetIndex)
			utils.Checkerr(err, "")
			if nIndex >= len(xlFile.Sheets) {
				fmt.Printf("Sheet index should small then length of sheets.sheetIndex:%d, "+
					"len(xlFile.Sheets):%d\n", sheetIndex, len(xlFile.Sheets))
				return
			}
			sheet_ := xlFile.Sheets[nIndex]
			fmt.Printf("--------sheetIndex%s, %s----------\n", sheetIndex, sheet_.Name)
			err = ConvertFromExcel(c, sheet_, db, dataStartRow, driverName, tableName)
			if err != nil && utils.Checkerr2(err, fmt.Sprintf("sheetIndex:%d", nIndex)) {
				msg := "error"
				if err != nil {
					msg += ":" + err.Error()
				}
				sign <- msg
			} else {
				sign <- "success"
			}
		}(sheetIndex)
	}
	for sheetIndex := 0; sheetIndex < len(sheetSlice); sheetIndex++ {
		msg := <-sign
		fmt.Printf("sheetIndex:%d, msg:%s\n", sheetIndex, msg)
	}
}
