package snmp

import (
	"fmt"
	"slices"
	"strings"
)

type TableMeta struct {
	Name    string
	Prefix  string
	Index   []MetricName
	Columns []*MetricMeta
}

type MetricFlag uint

const (
	MetricIsString MetricFlag = 1 << iota
	MetricIsPartOfIndex
)

type MetricMeta struct {
	Name        MetricName
	SnakeName   string
	SnmpType    string
	Help, Type  string
	DisplayHint string
	Formatter   ValueFormatFunc
	Enums       map[int]string
	TableMeta   *TableMeta
	Flags       MetricFlag
}

func (meta *MetricMeta) IsLabel() bool {
	return meta.Flags&(MetricIsString|MetricIsPartOfIndex) != 0
}

type RowIndex string

type MetricName string

type Value struct {
	Text    string
	Numeric bool
	//Value   asn1binary.Value
}

type MetricValues struct {
	Meta   *MetricMeta
	Values map[RowIndex]Value
}

func (mv *MetricValues) AddMetric(row RowIndex, value Value) error {
	//	other, exists := mv.Values[row]
	//	if exists {
	//		if reflect.DeepEqual(value, other) {
	//			return nil
	//		}
	//		//return fmt.Errorf("duplicate row %s: %v, %v", row, other, value)
	//	}
	mv.Values[row] = value
	return nil
}

type MetricBlock struct { //map of lists
	TableMeta   *TableMeta
	RowIndexes  []RowIndex
	MetricNames []MetricName
	Metrics     map[MetricName]*MetricValues
}

func (mb *MetricBlock) IsNewRow(index RowIndex) bool {
	if slices.Contains(mb.RowIndexes, index) {
		return false
	}
	mb.RowIndexes = append(mb.RowIndexes, index)
	return true
}

func (mb *MetricBlock) AddMetric(printer *MetricPrinter, meta *MetricMeta, row RowIndex, value Value) error {
	values, ok := mb.Metrics[meta.Name]
	if !ok {
		mb.MetricNames = append(mb.MetricNames, meta.Name)
		values = &MetricValues{
			Meta:   meta,
			Values: make(map[RowIndex]Value),
		}
		mb.Metrics[meta.Name] = values
	}
	err := values.AddMetric(row, value)
	return err
}

func (mb *MetricBlock) Init(tableMeta *TableMeta) {
	mb.TableMeta = tableMeta
	mb.Metrics = make(map[MetricName]*MetricValues)
	if mb.MetricNames == nil {
		mb.MetricNames = make([]MetricName, 0, 16)
		mb.RowIndexes = make([]RowIndex, 0, 16)
		return
	}
	mb.MetricNames = mb.MetricNames[:0]
	mb.RowIndexes = mb.RowIndexes[:0]
}

// findCommonPrefix finds the common prefix among metric names.
func findCommonPrefix(a, b string) string {
	prefixLen := min(len(a), len(b))
	for i := 0; i < prefixLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:prefixLen]
}

func (mb *MetricBlock) LabelMap() map[RowIndex]string {

	if mb.TableMeta == nil {
		return nil
	}

	prefixLen := len(mb.TableMeta.Prefix)

	labelMap := make(map[RowIndex]string)
	sb := &strings.Builder{}
	for _, index := range mb.RowIndexes {
		sb.Reset()
		for j, column := range mb.TableMeta.Columns {
			if column.IsLabel() {
				if j > 0 {
					sb.WriteString(",")
				}
				metrics := mb.Metrics[column.Name]
				if metrics == nil {
					continue
				}
				indexValue, ok := metrics.Values[index]
				columnName := column.SnakeName[prefixLen:]
				if ok {
					fmt.Fprintf(sb, "%s=%q", columnName, indexValue.Text)
				} else {
					fmt.Fprintf(sb, "%s=%q", columnName, "")
				}
			}
		}
		labelMap[index] = sb.String()
	}
	return labelMap
}
