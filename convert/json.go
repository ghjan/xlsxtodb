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
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
)

func FromJson(c *Columns, dataMap []map[string]interface{},
	db *sql.DB, driverName, tableName string, titles []StringValue) (err error) {
	for _, item := range titles {
		c.sourceColumns = append(c.sourceColumns, item)
	}

	c.ParseColumns()
	rowsNum := len(dataMap)
	for rowIndex, record := range dataMap {
		dbRow := &DBRow{value: make(map[string]string), sql: "", ot: OtherTable{}}
		tmp := 0
		// 字段
		var insertIntoFieldNames []string     //INSERT INTO ()
		needConflictOnFields := ""            //CONFLICT ON()
		distinctExcludedFieldSet := set.New() //where tbl.c3 is distinct from excluded.c3 or tbl.c4 is distinct from excluded.c4
		updatedFieldSet := set.New()          // DO UPDATE SET

		needConflictOnFieldsOther := ""
		updateSetSql := ""
		whereSql := ""
		//var excludedFields []string
		id := ""
		returningFields := []string{"id"}
		var rows *sql.Rows
		var uniqTogetherMap = map[string]string{}
		var values []string
		for key, columnFieldValues := range c.useColumns {
			if columnFieldValues == nil || len(dataMap) <= 0 {
				fmt.Printf("c.useColumns:%#v\n", c.useColumns)
				continue
			}
			if key >= len(record) || record[columnFieldValues[0]] == nil {
				dbRow.value[columnFieldValues[0]] = ""
			} else {
				dbRow.value[columnFieldValues[0]] = utils.GetStringValue(record[columnFieldValues[0]])
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
				if len(columnFieldValues) > 1 {
					switch columnFieldValues[1] {
					case "generate":
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
						//if id != "" {
						//	continue
						//}
						uniqueSql := "SELECT id, " + columnFieldValues[0] + " FROM  " +
							utils.EscapeString(driverName, tableName) + "  WHERE  " + columnFieldValues[0] + "  = '" +
							utils.EscapeSpecificChar(dbRow.value[columnFieldValues[0]]) + "'"
						//fmt.Printf("uniqueSql:%s\n", uniqueSql)
						result, _ := utils.FetchRow(db, uniqueSql)
						id = (*result)["id"]
						if id != "" {
							uniqTogetherMap[columnFieldValues[0]] = utils.EscapeSpecificChar(dbRow.value[columnFieldValues[0]])
							if needConflictOnFields == "" {
								needConflictOnFields = columnFieldValues[0]
							} else {
								needConflictOnFields += "," + columnFieldValues[0]
							}
						}
					case "uniq_together":
						uniqTogetherMap[columnFieldValues[0]] = utils.EscapeSpecificChar(dbRow.value[columnFieldValues[0]])
						if needConflictOnFields == "" {
							needConflictOnFields = columnFieldValues[0]
						} else {
							needConflictOnFields += "," + columnFieldValues[0]
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
						result, _ := utils.FetchRow(db, "SELECT  "+columnFieldValues[3]+"  FROM  "+
							utils.EscapeString(driverName, columnFieldValues[2])+"  WHERE "+columnFieldValues[4]+
							" = '"+dbRow.value[columnFieldValues[0]]+"'")
						if (*result)["id"] == "" {
							//sign <- "error"
							msg := "[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]表 " +
								columnFieldValues[2] + " 中没有找到 " + columnFieldValues[4] + " 为 " +
								dbRow.value[columnFieldValues[0]] + " 的数据，自动跳过"
							//fmt.Println(msg)
							err = errors.New(msg)
							return
						}
						distinctExcludedFieldSet.Add(columnFieldValues[0])
						updatedFieldSet.Add(columnFieldValues[0])
						dbRow.value[columnFieldValues[0]] = (*result)["id"]
					}
				}
				var pro bool
				dbRow.value[columnFieldValues[0]], pro = ParseValue(dbRow.value[columnFieldValues[0]])

				if dbRow.value[columnFieldValues[0]] != "" {
					insertIntoFieldNames = append(insertIntoFieldNames, columnFieldValues[0])
					values = append(values, utils.EscapeValuesString(driverName, dbRow.value[columnFieldValues[0]]))
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
		dbRow.sql = insertSql
		if needConflictOnFields != "" && updateSetSql != "" && whereSql != "" {
			dbRow.sql += " ON CONFLICT (" + needConflictOnFields + ") DO UPDATE SET " +
				updateSetSql + whereSql
		}
		dbRow.sql += " RETURNING id;"
		rows, err = db.Query(dbRow.sql)
		var idOfMainRecord int
		if err == nil {
			rows.Scan(&idOfMainRecord)
		} else {
			fmt.Printf("err:%s\n", err.Error())
			fmt.Printf("dbRow.sql:%s\n", dbRow.sql)
		}
		rows.Close()

		if idOfMainRecord == 0 {
			idOfMainRecord = int(dbRow.insertID)
			fmt.Println(dbRow.sql)
			if idOfMainRecord == 0 && id != "" {
				idOfMainRecord, _ = strconv.Atoi(id)
			}
		}
		//if idOfMainRecord == 0 {
		//	err = errors.New("id is 0")
		//	return
		//}
		utils.Checkerr(err, dbRow.sql)
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
			rows, err := db.Query(dbRow.ot.sql + ";")
			rows.Close()
			utils.Checkerr(err, dbRow.ot.sql)

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

func JsonToDB(db *sql.DB, driverName, tableName, jsonFileName string, titles []StringValue) (err error) {

	data, err := utils.ReadJson2Slice(jsonFileName)
	utils.Checkerr(err, jsonFileName)

	c := new(Columns)
	if titles == nil {
		for k, _ := range data[0] {
			titles = append(titles, StringValue{k})
		}
	}
	var dataCleaned []map[string]interface{}
	for _, record := range data {
		recordNew := make(map[string]interface{})
		for k, v := range record {
			newK := strings.Split(k, "|")[0]
			recordNew[newK] = v
		}
		dataCleaned = append(dataCleaned, recordNew)
	}
	c.tableColumnMap, err = GetTableColumns(db, driverName, tableName)
	utils.Checkerr(err, "")
	err = FromJson(c, dataCleaned, db, driverName, tableName, titles)
	utils.Checkerr(err, "")
	return
}
