package convert

import (
	"github.com/ghjan/xlsxtodb/pkg/set"
)

type StringValueInterface interface {
	String() string
}

type StringValue struct {
	Value string
}

func (sv StringValue) String() string {
	return sv.Value
}

//列
type Columns struct {
	sourceColumns  []StringValueInterface //xlsx/json中的列
	tableColumnMap map[int]string         //db中的列
	useColumns     map[int][]string       // 把xlsx中的列做解析 每个列名split为一个数组
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
	NeedNotUpdateFieldSet *set.StringSet
	NeedGenerateFieldSet  *set.StringSet
)

func init() {
	NeedNotUpdateFieldSet = set.NewFromSlice([]string{"created_at", "created", "created_on"})
	NeedGenerateFieldSet = set.NewFromSlice([]string{"uuid", "short_uuid",})

}
