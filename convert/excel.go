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

func FromExcel(c *Columns, sheet *xlsx.Sheet, db *sql.DB, dataStartRow int, driverName,
	tableName string, sheetIndex string, debug bool) (err error) {
	err = nil
	if sheet.Rows == nil {
		err = errors.New("该sheet没有数据")
		return
	}
	for _, cell := range sheet.Rows[0].Cells {
		c.sourceColumns = append(c.sourceColumns, cell)
	}

	c.ParseColumns()
	//ch := make(chan int, 50)
	rowsNum := len(sheet.Rows) //- dataStartRow + 2
OutFor:
	for rowIndex := dataStartRow - 1; rowIndex < rowsNum; rowIndex++ {
		//utils.Checkerr(err, "")
		dbRow := &DBRow{value: make(map[string]string), sql: "", ot: OtherTable{}}
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
		var resultUniqSql *map[string]string
	InnerFor:
		for key, columnFieldValues := range c.useColumns {
			if columnFieldValues == nil || len(sheet.Rows) <= 0 {
				fmt.Printf("c.useColumns:%#v\n", c.useColumns)
				continue
			}
			if key >= len(sheet.Rows[rowIndex].Cells) || sheet.Rows[rowIndex].Cells[key] == nil {
				dbRow.value[columnFieldValues[0]] = ""
			} else {
				dbRow.value[columnFieldValues[0]] = sheet.Rows[rowIndex].Cells[key].String()
			}
			//解析内容
			if columnFieldValues[0] == ":other" {
				dbRow.ot.value = strings.Split(dbRow.value[columnFieldValues[0]], "|")
				sqlOther := "SELECT * FROM " + utils.EscapeString(driverName, dbRow.ot.value[0])
				rows, err = db.Query(sqlOther)
				utils.Checkerr(err, sqlOther)
				dbRow.ot.columns, err = rows.Columns()
				rows.Close()
				utils.Checkerr(err, sqlOther)

				for _, v := range columnFieldValues {
					if strings.Index(v, "unique=") >= 0 {
						needConflictOnFieldsOther = strings.Join(strings.Split(strings.TrimPrefix(v,
							"unique="), "+"), ",")
					}
				}
			} else {
				isGenerate := false
				if len(columnFieldValues) > 1 {
					switch columnFieldValues[1] {
					case "generate":
						isGenerate = true
						if columnFieldValues[0] == "uuid" {
							dbRow.value[columnFieldValues[0]] = uuid.New().String()
							returningFields = append(returningFields, columnFieldValues[0])
						} else if columnFieldValues[0] == "short_uuid" {
							dbRow.value[columnFieldValues[0]] = shortuuid.New()
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
						uniqueSql := "SELECT * FROM  " +
							utils.EscapeString(driverName, tableName) + "  WHERE  " + columnFieldValues[0] + "  = '" +
							utils.EscapeSpecificChar(dbRow.value[columnFieldValues[0]]) + "'"
						if dbRow.value[columnFieldValues[0]] == "" {
							fmt.Printf("[sheet"+sheetIndex+"-"+strconv.Itoa(rowIndex+1)+"/"+strconv.Itoa(rowsNum+1)+
								"] FromExcel ignored:uniq field value is blank, uniqueSql:%s\n", uniqueSql)
							continue OutFor
						}
						//fmt.Printf("uniqueSql:%s\n", uniqueSql)
						resultUniqSql, _ = utils.FetchRow(db, uniqueSql)
						id = (*resultUniqSql)["id"]
						if id != "" && !strings.Contains(needConflictOnFields, columnFieldValues[0]) {
							if needConflictOnFields == "" {
								needConflictOnFields = columnFieldValues[0]
							} else {
								needConflictOnFields += "," + columnFieldValues[0]
							}
						}
					case "uniq_together":
						uniqTogetherMap[columnFieldValues[0]] = utils.EscapeSpecificChar(dbRow.value[columnFieldValues[0]])
						if !strings.Contains(needConflictOnFields, columnFieldValues[0]) {
							if needConflictOnFields == "" {
								needConflictOnFields = columnFieldValues[0]
							} else {
								needConflictOnFields += "," + columnFieldValues[0]
							}
						}
					case "password":
						tmpvalue := strings.Split(dbRow.value[columnFieldValues[0]], "|")
						if len(tmpvalue) == 2 {
							if []byte(tmpvalue[1])[0] == ':' {
								if _, ok := dbRow.value[string([]byte(tmpvalue[1])[1:])]; ok {
									dbRow.value[columnFieldValues[0]] = tmpvalue[0] +
										dbRow.value[string([]byte(tmpvalue[1])[1:])]
								} else {
									msg := "[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) +
										"]密码盐" + string([]byte(tmpvalue[1])[1:]) + "字段不存在，自动跳过"
									//fmt.Println(msg)
									err = errors.New(msg)
									//sign <- "error"
									return
								}
							} else {
								dbRow.value[columnFieldValues[0]] += tmpvalue[1]
							}
						} else {
							dbRow.value[columnFieldValues[0]] = tmpvalue[0]
						}
						switch columnFieldValues[2] {
						case "md5":
							dbRow.value[columnFieldValues[0]] = string(md5.New().Sum([]byte(dbRow.value[columnFieldValues[0]])))
						case "bcrypt":
							pass, _ := bcrypt.GenerateFromPassword([]byte(dbRow.value[columnFieldValues[0]]), 13)
							dbRow.value[columnFieldValues[0]] = string(pass)
						}
					case "find":
						if strings.TrimSpace(dbRow.value[columnFieldValues[0]]) == "" {
							//空白列 跳过该列
							continue InnerFor
						}
						sqlFind := "SELECT * FROM  " + utils.EscapeString(driverName, columnFieldValues[2]) +
							"  WHERE " + columnFieldValues[4] + " = '" +
							strings.TrimSpace(dbRow.value[columnFieldValues[0]]) + "'"
						result, _ := utils.FetchRow(db, sqlFind)
						if (*result)[columnFieldValues[3]] == "" {
							msg := "[sheet" + sheetIndex + "-" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]表" +
								columnFieldValues[2] + "中没有找到" + columnFieldValues[4] + "为" +
								dbRow.value[columnFieldValues[0]] + "的数据，自动跳过"
							fmt.Println(msg)
							//有的数据不合法 跳过就可以 不必停止处理其它数据
							continue OutFor
						}
						distinctExcludedFieldSet.Add(columnFieldValues[0])
						updatedFieldSet.Add(columnFieldValues[0])
						dbRow.value[columnFieldValues[0]] = (*result)[columnFieldValues[3]]
					}

					switch columnFieldValues[len(columnFieldValues)-1] {
					case "uniq_together":
						uniqTogetherMap[columnFieldValues[0]] = utils.EscapeSpecificChar(dbRow.value[columnFieldValues[0]])
						if !strings.Contains(needConflictOnFields, columnFieldValues[0]) {
							if needConflictOnFields == "" {
								needConflictOnFields = columnFieldValues[0]
							} else {
								needConflictOnFields += "," + columnFieldValues[0]
							}
						}
					}
				}
				var processed bool
				dbRow.value[columnFieldValues[0]], processed = ParseValue(dbRow.value[columnFieldValues[0]])

				if dbRow.value[columnFieldValues[0]] != "" {
					insertIntoFieldNames = append(insertIntoFieldNames, columnFieldValues[0])
					values = append(values, utils.EscapeValuesString(driverName, strings.TrimSpace(dbRow.value[columnFieldValues[0]])))
					if !isGenerate { //自动生成的不能放在update语句里面
						updatedFieldSet.Add(columnFieldValues[0])
					}
					if !processed {
						distinctExcludedFieldSet.Add(columnFieldValues[0])
					}
					tmp++
				}
			}
		}
		var result *map[string]string
		uniqTogetherSql := ""
		if id == "" {
			if len(uniqTogetherMap) > 0 {
				uniqTogetherSql = " select * from " + tableName + " where "
				indexTemp := 0
				for field, value := range uniqTogetherMap {
					if indexTemp > 0 {
						uniqTogetherSql += " and "
					}
					uniqTogetherSql += field + "=" + utils.EscapeValuesString(driverName, value)
					indexTemp += 1
				}
				result, err = utils.FetchRow(db, uniqTogetherSql)
				if err != nil {
					fmt.Printf("FetchRow err,uniqTogetherSql:%s, error:%s\n", uniqTogetherSql, err.Error())
				}
			}
		} else {
			result = resultUniqSql
		}
		if result != nil {
			for _, fieldReturning := range returningFields {
				if fieldReturning == "id" {
					id = (*result)["id"]
				} else { // 如果没有值，就需要update
					if (*result)[fieldReturning] == "" || (*result)[fieldReturning] == "NULL" {
						updatedFieldSet.Add(fieldReturning)
						distinctExcludedFieldSet.Add(fieldReturning)
						fmt.Printf("fieldReturning:%s, (*result)[fieldReturning]:%s, uniqTogetherSql:%s\n", fieldReturning, (*result)[fieldReturning], uniqTogetherSql)
					} else {
						updatedFieldSet.Remove(fieldReturning)
						distinctExcludedFieldSet.Remove(fieldReturning)
					}
				}
			}
		}

		insertSql, updateSetSql, whereSql := GetUpdateSql(driverName, tableName, insertIntoFieldNames, values,
			needConflictOnFields, updatedFieldSet, distinctExcludedFieldSet)
		dbRow.sql = insertSql
		useUpdate := false
		if needConflictOnFields != "" && updateSetSql != "" && whereSql != "" {
			dbRow.sql += " ON CONFLICT (" + needConflictOnFields + ") DO UPDATE SET " +
				updateSetSql + whereSql
			useUpdate = true
		}
		dbRow.sql += " RETURNING id"
		if debug {
			fmt.Printf("dbRow.sql:%s\n", dbRow.sql)
		}
		err = db.QueryRow(dbRow.sql + ";").Scan(&dbRow.insertID)
		if err != nil || dbRow.insertID == 0 {
			if !useUpdate || strings.Index(err.Error(), "no rows in result set") < 0 {
				fmt.Printf("err:%s\n", err.Error())
				fmt.Printf("dbRow.sql:%s\n", dbRow.sql)
			} else if strings.Index(err.Error(), "no rows in result set") >= 0 {
				err = nil
			}
		}
		idOfMainRecord := int(dbRow.insertID)
		ignored := idOfMainRecord == 0
		if ignored {
			fmt.Printf("ignored,sql:%s\n", dbRow.sql)
			if id != "" {
				idOfMainRecord, _ = strconv.Atoi(id)
			}
		} else if debug {
			fmt.Printf("debug,sql:%s\n", dbRow.sql)
		}
		if idOfMainRecord == 0 {
			err = errors.New("id is 0")
			return
		}
		utils.Checkerr(err, dbRow.sql)
		if idOfMainRecord > 0 && err == nil {
			//更新id自增长值导入数据成功
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
		if dbRow.ot.value != nil {
			tableNameOther = dbRow.ot.value[0]
			tmp = 0
			var fieldNames []string
			var values []string
			for key, value := range dbRow.ot.columns {
				var pro bool
				dbRow.ot.value[key+1], pro = ParseValue(dbRow.ot.value[key+1])
				if dbRow.ot.value[key+1] == ":id" {
					dbRow.ot.value[key+1] = strconv.Itoa(idOfMainRecord)
				}
				if dbRow.ot.value[key+1] != "" {
					if value != "" {
						fieldNames = append(fieldNames, value)
						values = append(values, dbRow.ot.value[key+1])
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
			dbRow.ot.sql = insertSql
			if needConflictOnFieldsOther != "" {
				dbRow.ot.sql += " ON CONFLICT (" + needConflictOnFieldsOther + ") DO UPDATE SET " +
					updateSetSql + whereSql
			}
			if debug {
				fmt.Printf("dbRow.ot.sql:%s\n", dbRow.ot.sql)
			}
			rows, err := db.Query(dbRow.ot.sql + ";")
			rows.Close()
			utils.Checkerr(err, dbRow.ot.sql)

		}
		fmt.Println("[sheet" + sheetIndex + "-" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]导入数据成功")
		//sign <- "success"
		//}//end of go func
	}
	//for rowIndex := dataStartRow - 1; rowIndex <= rowsNum; rowIndex++ {
	//	<-sign
	//}
	return
}

func ExcelToDB(db *sql.DB, driverName, tableName, excelFileName, sheets string, dataStartRow int, debug bool) {
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
			err = FromExcel(c, sheet_, db, dataStartRow, driverName, tableName, sheetIndex, debug)
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
