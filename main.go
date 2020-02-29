// xlsxtodb project main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/ghjan/xlsxtodb/convert"
	"github.com/ghjan/xlsxtodb/pkg/utils"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"os"
	"runtime"
)

var (
	// 数据库驱动器名字
	driverName string
	// dsn
	dsn string

	// 对应的数据库表名
	tableName string

	//对应的excel文件名
	excelFileName string
	//对应的json文件名
	jsonFileName string
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
	flag.StringVar(&jsonFileName, "json", "", "json 对应的json文件名")

	// 从arguments中解析注册的flag。必须在所有flag都注册好而未访问其值时执行。未注册却使用flag -help时，会返回ErrHelp。
	flag.Parse()
	// 打印
	if driverName == "" || tableName == "" || ((excelFileName == "" || sheets == "") && jsonFileName == "") ||
		(dsn == "" && (username == "" || host == "" || port == "" || dbname == "")) {
		fmt.Println("请按照格式输入（excel-》db)：xlsxtodb -driver [mysql/postgres] -dsn [DSN] -table [tablename] -excel [xlsx文件名] -sheets [sheetIndex,sheetIndex] -u username -p password -h host -P port -d databasename")
		fmt.Println("或者（json-》db)：xlsxtodb -driver [mysql/postgres] -dsn [DSN] -table [tablename] -json [json文件名]  -u username -p password -h host -P port -d databasename")
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
	if excelFileName != "" {
		fmt.Printf("driverName=%v dsn=%v table=%v excel=%v\n", driverName, dsn, tableName, excelFileName)
	} else {
		fmt.Printf("driverName=%v dsn=%v table=%v json=%v\n", driverName, dsn, tableName, jsonFileName)

	}

}
func connectDB() *sql.DB {
	db, err := sql.Open(driverName, dsn)
	utils.Checkerr(err, "sql.Open(driverName, dsn),"+driverName+","+dsn)

	err = db.Ping()
	utils.Checkerr(err, driverName+","+dsn)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(30)
	return db
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	db := connectDB()

	defer db.Close()
	if (excelFileName != "") {
		convert.ExcelToDB(db, driverName, tableName, excelFileName, sheets, dataStartRow, false)
	} else {
		convert.JsonToDB(db, driverName, tableName, jsonFileName, nil)
	}

}
