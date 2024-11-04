package snmp

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type TableMeta struct {
	Name    string
	Index   []MetricName
	Columns []*MetricMeta
}

type MetricMeta struct {
	Name        MetricName
	SnmpType    string
	Help, Type  string
	DisplayHint string
	Formatter   ValueFormatFunc
	Enums       map[int]string
	TableMeta   *TableMeta
}

type RowIndex string

type MetricName string

type MetricValues struct {
	Meta   *MetricMeta
	Values map[RowIndex]Value
}

type Value struct {
	Text    string
	Numeric bool
	//Value   asn1binary.Value
}

func (mv *MetricValues) AddMetric(row RowIndex, value Value) error {
	other, exists := mv.Values[row]
	if exists {
		if reflect.DeepEqual(value, other) {
			return nil
		}
		//return fmt.Errorf("duplicate row %s: %v, %v", row, other, value)
	}
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

func (mb *MetricBlock) LabelMap() map[RowIndex]string {

	if mb.TableMeta == nil {
		return nil
	}

	prefix := mb.MetricNames[0]
	for i := 1; i < len(mb.MetricNames); i++ {
		name := mb.MetricNames[i]
		//find the common prefix
		for j := 0; j < len(prefix); j++ {
			if prefix[j] != name[j] {
				prefix = prefix[:j]
				break
			}
		}
	}

	labelMap := make(map[RowIndex]string)
	for _, index := range mb.RowIndexes {
		sb := &strings.Builder{}
		for j, columnName := range mb.TableMeta.Index {
			indexValues := mb.Metrics[columnName]
			if indexValues != nil {
				if j > 0 {
					sb.WriteString(",")
				}
				indexValue, ok := indexValues.Values[index]
				columnName = columnName[len(prefix):]
				if ok {
					fmt.Fprintf(sb, "%s=%q", columnName, indexValue.Text)
				} else {
					fmt.Fprintf(sb, "%s=%q", columnName, "")
				}
			} else {
				fmt.Printf("debug unknown column %s\n", columnName)
			}
		}
		labelMap[index] = sb.String()
	}
	return labelMap
}
