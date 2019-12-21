package main

import (
	"fmt"
	"github.com/ghjan/xlsxtodb/custom/excelstruct"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/tealeg/xlsx"
	"time"
)

/*
原文链接：https://blog.csdn.net/qq_25504271/article/details/79121598
*/
type ExcelData interface {
	CreateMaps() []map[string]string
	ChangeTime(source string) time.Time
}

type Good struct {
	ID           uint `gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `sql:"index"`
	GoodName     string     `gorm:"type:varchar(50)"`
	GoodMainImg  string
	GoodDescLink string
	CategoryName string
	TaobaokeLink string
	GoodPrice    float64
}

var DB *gorm.DB

func main() {
	gormDB, err := gorm.Open("mysql", "root:123456@/taobao?charset=utf8&parseTime=True&loc=Local")
	gormDB.LogMode(true)
	gormDB.AutoMigrate(&Good{})
	gormDB.Model(&Good{}).AddUniqueIndex("idx_goodname", "good_name")

	//gormDB.SingularTable(true)
	DB = gormDB
	defer gormDB.Close()
	if err != nil {
		panic(err)
	}

	es := excelstruct.ExcelStrcut{}
	temp := Good{}
	es.Model = temp
	info := es.ReadExcel("./test.xlsx", "Sheet1").CreateMaps()
	for i := 0; i < len(info); i++ {
		es.SaveDb(DB, info[i], &temp, "good_name")
		temp = Good{}
	}
}

func tranverseExcelfile(excelFileName string) {
	fmt.Printf("excelFileName:%s\n", excelFileName)
	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		fmt.Printf("open failed: %s\n", err)
	}
	for _, sheet := range xlFile.Sheets {
		fmt.Printf("Sheet Name: %s\n", sheet.Name)
		for _, row := range sheet.Rows {
			for _, cell := range row.Cells {
				text := cell.String()
				fmt.Printf("%s ", text)
			}
			fmt.Println("")
		}
	}
}
