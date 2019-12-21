// xlsxtodb project main.go
package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/ghjan/xlsxtodb/pkg/set"
	"github.com/ghjan/xlsxtodb/pkg/utils"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tealeg/xlsx"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

//列
type Columns struct {
	xlsxColumns    []*xlsx.Cell
	tableColumnMap map[int]string
	useColumns     map[int][]string
}

//行
type Row struct {
	insertID int64
	sql      string
	value    map[string]string
	ot       OtherTable
}

//附表
type OtherTable struct {
	sql     string
	value   []string
	columns []string
}

var (
	// 数据库驱动器名字
	driverName string
	// dsn
	dsn string

	// 对应的数据库表名
	tableName string

	//对应的excel文件名
	excelFileName string
	//对应的excel sheet index, begin from 0
	sheets string
	//对应的excel dataStartRow, begin from 1
	dataStartRow int

	//有些情况不提供dsn 需要提供下列的信息
	username string
	password string
	host     string
	port     string
	dbname   string
)

var (
	NeedNotUpdateFieldSet *set.StringSet
)

func init() {
	NeedNotUpdateFieldSet = set.NewFromSlice([]string{"created_at", "created", "created_on"})
}

func init() {
	/*
	   定义变量接收控制台参数
	*/
	// StringVar用指定的名称、控制台参数项目、默认值、使用信息注册一个string类型flag，并将flag的值保存到p指向的变量
	flag.StringVar(&driverName, "driver", "mysql", "数据库驱动器名字，mysql")
	flag.StringVar(&dsn, "dsn", "", "dsn 数据连接字符串 root:123@tcp(127.0.0.1:3306)/dbname")
	flag.StringVar(&tableName, "table", "", "table 对应的数据库表名")
	flag.StringVar(&excelFileName, "excel", "", "excel 对应的excel文件名")
	flag.StringVar(&sheets, "sheets", "0", "用逗号隔开的sheets，每个对应excel sheet index, begin from 0")
	flag.IntVar(&dataStartRow, "dataStartRow", 2, "对应的excel dataStartRow, begin from 1")
	flag.StringVar(&username, "u", "", "用户名,默认为空")
	flag.StringVar(&password, "p", "", "密码,默认为空")
	flag.StringVar(&host, "h", "127.0.0.1", "主机名,默认 127.0.0.1")
	flag.StringVar(&port, "P", "3306", "端口号,3306")
	flag.StringVar(&dbname, "d", "", "数据库名称")

	// 从arguments中解析注册的flag。必须在所有flag都注册好而未访问其值时执行。未注册却使用flag -help时，会返回ErrHelp。
	flag.Parse()
	// 打印
	if driverName == "" || tableName == "" || excelFileName == "" || (dsn == "" && (username == "" || host == "" || port == "" || dbname == "" || sheets == "")) {
		fmt.Println("请按照格式输入：xlsxtodb -driver [mysql/postgres] -dsn [DSN] -table [tablename] -excel [xlsx文件名] -sheets [sheetIndex,sheetIndex] -u username -p password -h host -P port -d databasename")
		os.Exit(-1)
	}
	if dsn == "" {
		if driverName == "postgres" {
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				username, password, host, port, dbname)
		} else if driverName == "mysql" {
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
				username, password, host, port, dbname)

		}
	}
	fmt.Printf("driverName=%v dsn=%v table=%v excel=%v\n", driverName, dsn, tableName, excelFileName)

}
func connectDB() *sql.DB {
	db, err := sql.Open(driverName, dsn)
	utils.Checkerr(err, driverName+","+dsn)

	err = db.Ping()
	utils.Checkerr(err, driverName+","+dsn)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)
	return db
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	db := connectDB()

	defer db.Close()
	c, _ := getColumns(db)

	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		utils.Checkerr(err, excelFileName)
	}
	sheetSlice := strings.Split(sheets, ",")
	for _, sheetIndex := range sheetSlice {
		nIndex, err := strconv.Atoi(sheetIndex)
		utils.Checkerr(err, "")
		if nIndex >= len(xlFile.Sheets) {
			fmt.Printf("Sheet index should small then length of sheets.sheet:%d, len(xlFile.Sheets):%d\n", sheetIndex, len(xlFile.Sheets))
			return
		}
		sheet_ := xlFile.Sheets[nIndex]
		fmt.Printf("--------sheet%s, %s----------\n", sheetIndex, sheet_.Name)
		err = convert(c, sheet_, db)
		utils.Checkerr(err, fmt.Sprintf("sheetIndex:%d", nIndex))
	}

}

func getColumns(db *sql.DB) (*Columns, error) {
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

func convert(c *Columns, sheet *xlsx.Sheet, db *sql.DB) (err error) {
	err = nil
	if sheet.Rows == nil {
		err = errors.New("该sheet没有数据")
		return
	}
	c.xlsxColumns = sheet.Rows[0].Cells
	c.paraseColumns()
	//ch := make(chan int, 50)
	sign := make(chan string, len(sheet.Rows))
	rowsNum := len(sheet.Rows) - dataStartRow + 2
	for i := dataStartRow - 1; i <= rowsNum; i++ {
		//ch <- i
		go func(rowIndex int) {
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
			var ids string
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
							needConflictOnFieldsOther = strings.Join(strings.Split(strings.TrimPrefix(v, "unique="), "+"), ",")
						}
					}
				} else {
					if len(value) > 1 {
						switch value[1] {
						case "unique":
							uniqueSql := "SELECT group_concat(id) as ids, count(" + value[0] + ") as has FROM  " +
								utils.EscapeString(driverName, tableName) + "  WHERE  " + value[0] + "  = '" +
								utils.EscapeSpecificChar(r.value[value[0]]) + "'"
							//fmt.Printf("uniqueSql:%s\n", uniqueSql)
							result, _ := utils.FetchRow(db, uniqueSql)
							has, _ := strconv.Atoi((*result)["has"])
							ids = (*result)["ids"]
							if ids != "" && ids != "{}" {
								ids = strings.TrimSuffix(strings.TrimPrefix(ids, "{"), "}")
							}
							if has > 0 {
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
										fmt.Print("[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]密码盐" + string([]byte(tmpvalue[1])[1:]) + "字段不存在，自动跳过\n")
										sign <- "error"
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
								fmt.Print("[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]表 " + value[2] + " 中没有找到 " + value[4] + " 为 " + r.value[value[0]] + " 的数据，自动跳过\n")
								sign <- "error"
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
			insertSql, updateSetSql, whereSql := getUpdateSql(tableName, fieldNames, values, needConflictOnFields,
				updatedFieldSet, excludedFieldSet)
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
				if ids != "" {
					idOfMainRecord, _ = strconv.Atoi(strings.Split(ids, ",")[0])
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
				insertSql, updateSetSql, whereSql = getUpdateSql(tableNameOther, fieldNames, values, needConflictOnFieldsOther, updatedFieldSet, excludedFieldSet)
				r.ot.sql = insertSql
				if needConflictOnFieldsOther != "" {
					r.ot.sql += " ON CONFLICT (" + needConflictOnFieldsOther + ") DO UPDATE SET " +
						updateSetSql + whereSql
				}

				db.QueryRow(r.ot.sql + ";")
			}
			fmt.Println("[" + strconv.Itoa(rowIndex+1) + "/" + strconv.Itoa(rowsNum+1) + "]导入数据成功")
			sign <- "success"
		}(i) //end of go func
	}
	for i := dataStartRow - 1; i <= rowsNum; i++ {
		<-sign
	}
	return
}

func getUpdateSql(tableName string, fieldNames []string, values []string, needConflictOnFields string, updatedFieldSet *set.StringSet, excludedFieldSet *set.StringSet) (sql string, updateSetSql string, whereSql string) {
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
func (c *Columns) paraseColumns() {
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
		result = strings.Replace(substr(base64.StdEncoding.EncodeToString(utils.Krand(32, utils.KC_RAND_KIND_ALL)), 0, 32), "+/", "_-", -1)
		processed = true
	default:
		result = val
		processed = false
	}
	return result, processed
}

//按长度截取字符串
func substr(str string, start int, length int) string {
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
