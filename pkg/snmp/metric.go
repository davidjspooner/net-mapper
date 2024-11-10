package snmp

import (
	"fmt"
	"slices"
	"strings"
)

type Table struct {
	prefix     string
	metricMeta *MetricMeta
	index      []MetricName
	columns    []*MetricMeta
}

type MetricFlag uint

const (
	MetricIsString MetricFlag = 1 << iota
	MetricIsPartOfIndex
)

type MetricMeta struct {
	name        MetricName
	snakeName   string
	snmpType    string
	help, Type  string
	displayHint string
	enums       map[int]string
	table       *Table
	flags       MetricFlag
}

func (meta *MetricMeta) IsLabel() bool {
	return meta.flags&(MetricIsString|MetricIsPartOfIndex) != 0
}

type RowIndex string

type MetricName string

type Value struct {
	text    string
	numeric bool
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
	table       *Table
	rowIndexes  []RowIndex
	metricNames []MetricName
	metrics     map[MetricName]*MetricValues
}

func (mb *MetricBlock) IsNewRow(index RowIndex) bool {
	if slices.Contains(mb.rowIndexes, index) {
		return false
	}
	mb.rowIndexes = append(mb.rowIndexes, index)
	return true
}

func (mb *MetricBlock) AddMetric(printer *MetricPrinter, meta *MetricMeta, row RowIndex, value Value) error {
	values, ok := mb.metrics[meta.name]
	if !ok {
		mb.metricNames = append(mb.metricNames, meta.name)
		values = &MetricValues{
			Meta:   meta,
			Values: make(map[RowIndex]Value),
		}
		mb.metrics[meta.name] = values
	}
	err := values.AddMetric(row, value)
	return err
}

func (mb *MetricBlock) Init(tableMeta *Table) {
	mb.table = tableMeta
	mb.metrics = make(map[MetricName]*MetricValues)
	if mb.metricNames == nil {
		mb.metricNames = make([]MetricName, 0, 16)
		mb.rowIndexes = make([]RowIndex, 0, 16)
		return
	}
	mb.metricNames = mb.metricNames[:0]
	mb.rowIndexes = mb.rowIndexes[:0]
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

	if mb.table == nil {
		return nil
	}

	prefixLen := len(mb.table.prefix)

	labelMap := make(map[RowIndex]string)
	sb := &strings.Builder{}
	for _, index := range mb.rowIndexes {
		sb.Reset()
		for j, column := range mb.table.columns {
			if column.IsLabel() {
				if j > 0 {
					sb.WriteString(",")
				}
				metrics := mb.metrics[column.name]
				if metrics == nil {
					continue
				}
				indexValue, ok := metrics.Values[index]
				columnName := column.snakeName[prefixLen:]
				if ok {
					fmt.Fprintf(sb, "%s=%q", columnName, indexValue.text)
				} else {
					fmt.Fprintf(sb, "%s=%q", columnName, "")
				}
			}
		}
		labelMap[index] = sb.String()
	}
	return labelMap
}
