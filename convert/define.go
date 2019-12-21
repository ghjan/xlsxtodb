package convert

import (
	"github.com/ghjan/xlsxtodb/pkg/set"
	"github.com/tealeg/xlsx"
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
	NeedNotUpdateFieldSet *set.StringSet
)

func init() {
	NeedNotUpdateFieldSet = set.NewFromSlice([]string{"created_at", "created", "created_on"})
}
