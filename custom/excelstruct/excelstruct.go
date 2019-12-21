package excelstruct

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/jinzhu/gorm"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ExcelStrcut struct {
	temp  [][]string
	Model interface{}
	Info  []map[string]string
}

func InStringSlice(haystack []string, needle string) int {
	for index, e := range haystack {
		if strings.TrimSpace(strings.ToLower(e)) == strings.TrimSpace(strings.ToLower(needle)) {
			return index
		}
	}

	return -1
}

func InIntSlice(haystack []int, needle int) int {
	for index, e := range haystack {
		if e == needle {
			return index
		}
	}

	return -1
}

func (excel *ExcelStrcut) CreateMaps() []map[string]string {

	var fieldKeyMap = make(map[string]int)
	//利用反射得到字段名
	for j, v := range excel.temp {
		//将数组  转成对应的 map
		var info = make(map[string]string)
		for i := 0; i < reflect.ValueOf(excel.Model).NumField(); i++ {
			obj := reflect.TypeOf(excel.Model).Field(i)
			if j == 0 { //第一行是field name
				keyIndex := InStringSlice(v, obj.Name)
				if keyIndex >= 0 {
					fieldKeyMap[obj.Name] = keyIndex
				}
			} else {

				if _, ok := fieldKeyMap[obj.Name]; ok {
					info[obj.Name] = v[fieldKeyMap[obj.Name]]
				} else {
					fmt.Printf("field:%s has no associated field in excel file\n", obj.Name)
				}
			}
			//if reflect.TypeOf(obj).Kind() == reflect.Struct{
			//	fmt.Printf("key:%s--val:%s\n",obj.Name,v[i])
			//}else{
			//	info[obj.Name] = v[i]
			//}

		}
		if len(info) > 0 {
			excel.Info = append(excel.Info, info)
		}

	}

	return excel.Info

}

func (excel *ExcelStrcut) ChangeTime(source string) time.Time {
	ChangeAfter, err := time.Parse("2006-01-02", source)
	if err != nil {
		log.Fatalf("转换时间错误:%s", err)
	}
	return ChangeAfter
}
func DeepCopy(src interface{}) (dst interface{}, err error) {
	var buf bytes.Buffer
	dst = nil
	if err = gob.NewEncoder(&buf).Encode(src); err != nil {
		fmt.Printf("DeepCopy, err:%s", err.Error())
		return
	}
	dst = gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
	return
}

func (excel *ExcelStrcut) SaveDb(DB *gorm.DB, info map[string]string, temp interface{}, uniqField string) *ExcelStrcut {
	t := reflect.ValueOf(temp).Elem()
	var uniqFieldValue string
	for k, v := range info {

		//fmt.Println(t.FieldByName(k).t.FieldByName(k).Kind())
		//fmt.Println("key:%v---val:%v",t.FieldByName(k),t.FieldByName(k).Kind())
		var tempV string
		switch t.FieldByName(k).Kind() {
		case reflect.String:
			t.FieldByName(k).Set(reflect.ValueOf(v))
			tempV = v
		case reflect.Float32:
			tempV, err := strconv.ParseFloat(v, 32)
			if err != nil {
				log.Printf("string to Float32 err：%v", err)
			}
			t.FieldByName(k).Set(reflect.ValueOf(tempV))
		case reflect.Float64:
			tempV, err := strconv.ParseFloat(v, 64)
			if err != nil {
				log.Printf("string to float64 err：%v", err)
			}

			t.FieldByName(k).Set(reflect.ValueOf(tempV))
		case reflect.Uint64:
			reflect.ValueOf(v)
			tempV, err := strconv.ParseUint(v, 0, 64)
			if err != nil {
				log.Printf("string to uint64 err：%v", err)
			}
			t.FieldByName(k).Set(reflect.ValueOf(tempV))
		case reflect.Int:
			reflect.ValueOf(v)
			tempV, err := strconv.ParseInt(v, 0, 64)
			if err != nil {
				log.Printf("string to Int err：%v", err)
			}
			t.FieldByName(k).Set(reflect.ValueOf(tempV))
		case reflect.Uint32:
			reflect.ValueOf(v)
			tempV, err := strconv.ParseUint(v, 0, 32)
			if err != nil {
				log.Printf("string to Uint32 err：%v", err)
			}
			t.FieldByName(k).Set(reflect.ValueOf(tempV))
		case reflect.Bool:
			reflect.ValueOf(v)
			tempV, err := strconv.ParseBool(v)
			if err != nil {
				log.Printf("string to bool err：%v", err)
			}
			t.FieldByName(k).Set(reflect.ValueOf(tempV))

		case reflect.Struct:
			tempV, err := time.Parse("2006-01-02", v)
			if err != nil {
				log.Fatalf("string to time err:%v", err)
			}
			t.FieldByName(k).Set(reflect.ValueOf(tempV))
		default:
			fmt.Println("type err")

		}
		if SameFieldName(strings.TrimSpace(k), strings.TrimSpace(uniqField)) {
			uniqFieldValue = string(tempV)
		}

	}
	//err := DB.Create(temp).Error

	err := DB.Where(uniqField+" = ?", uniqFieldValue).Assign(temp).FirstOrCreate(temp).Error

	if err != nil {
		log.Fatalf("save temp table err:%v", err)
	}
	fmt.Printf("导入临时表成功")

	return excel
}
func SameFieldName(fieldName, fieldName2 string) (result bool) {
	result = fieldName == fieldName2
	if !result {
		if strings.Index(fieldName, "_") < 0 && strings.Index(fieldName2, "_") >= 0 {
			names := strings.Split(fieldName2, "_")
			camelName := ""
			for _, name := range names {
				camelName += strings.ToUpper(string(name[0])) + name[1:]
			}
			result = fieldName == camelName
		} else if strings.Index(fieldName, "_") >= 0 && strings.Index(fieldName2, "_") < 0 {
			return SameFieldName(fieldName2, fieldName)
		}
	}
	return
}

func (excel *ExcelStrcut) ReadExcel(file string, sheetName string) *ExcelStrcut {

	xlsx, err := excelize.OpenFile(file)
	if err != nil {
		fmt.Println("read excel:", err)
	}

	rows := xlsx.GetRows(sheetName)
	excel.temp = rows

	return excel

}
