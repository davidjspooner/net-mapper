package snmp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibdb"
)

type MetricPrinter struct {
	w               io.Writer
	db              *mibdb.Database
	lastPrintedMeta *MetricMeta

	metricBlock   MetricBlock
	headerPrinted bool
}

var _ VarBindHandler = &MetricPrinter{}

func NewMetricPrinter(w io.Writer, db *mibdb.Database) *MetricPrinter {
	mp := &MetricPrinter{w: w, db: db}
	mp.metricBlock.Init(nil)
	return mp
}

func (printer *MetricPrinter) Handle(ctx context.Context, vb *VarBind) error {

	if len(vb.OID) > 0 && vb.OID[0] == 0 {
		vb.OID = vb.OID[1:]
	}

	//if vb.OID.String() == "1.3.6.1.2.1.3.1.1.1.3.1.169.254.18.13" {
	//	fmt.Println("debug vb.OID:", vb.OID)
	//}

	branch, index := printer.db.FindOID(vb.OID)
	if branch == nil || branch.Object() == nil {
		printer.db.Logger().WarnContext(ctx, "OID not found", slog.String("OID", vb.OID.String()))
		return nil
	}

	metaParent := printer.MetaDataForBranch(branch.Parent())
	meta := printer.MetaDataForBranch(branch)

	if metaParent.table == nil && len(index) > 1 {
		var oid asn1go.OID
		oid = append(oid, branch.Object().OID()...)
		oid = append(oid, index[0])
		printer.db.Logger().WarnContext(ctx, "Oid not known", slog.String("OID", oid.String()+".*"))
		return nil
	}
	s, _, err := DecodeValue(printer.db, &vb.Value)
	if err != nil {
		printer.db.Logger().WarnContext(ctx, "Error decoding value", slog.String("OID", vb.OID.String()), slog.String("error", err.Error()))
		return err
	}
	if strings.Contains(meta.displayHint, "x:") {
		sb := strings.Builder{}
		for i, b := range []byte(s) {
			if i > 0 {
				sb.WriteString(":")
			}
			sb.WriteString(fmt.Sprintf("%02x", b))
		}
		s = sb.String()
	} else if meta.displayHint == "e" {
		n, _ := strconv.Atoi(s)
		var ok bool
		s, ok = meta.enums[n]
		if !ok {
			s = fmt.Sprintf("%d", n)
		}
	} else if !strings.Contains(meta.displayHint, "a") {
		err = printer.queueLine(ctx, meta, metaParent.table, index, Value{s, true})
		return err
	}
	err = printer.queueLine(ctx, meta, metaParent.table, index, Value{s, false})
	return err
}

func (printer *MetricPrinter) queueLine(ctx context.Context, meta *MetricMeta, table *Table, index asn1go.OID, value Value) error {
	if (printer.lastPrintedMeta != meta) && (printer.metricBlock.table != table || table == nil) {
		err := printer.Flush(ctx)
		if err != nil {
			return err
		}
		printer.metricBlock.Init(table)
		printer.lastPrintedMeta = meta
	}

	row := RowIndex(index.String())

	if printer.metricBlock.IsNewRow(row) && printer.metricBlock.table != nil {
		tail := index
		var err error
		var value2 Value
		for _, columnName := range printer.metricBlock.table.index {
			def := printer.db.LookupName(string(columnName))
			if def != nil {
				obj, _ := def.(*mibdb.Object)
				if obj != nil {
					value2, tail, err = printer.Unmarshal(obj, tail)
					if err != nil {
						return err
					}
					index = tail
					meta2 := printer.MetaDataForObject(obj, nil)
					err := printer.metricBlock.AddMetric(printer, meta2, row, value2)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	err := printer.metricBlock.AddMetric(printer, meta, row, value)
	//err := printer.printLine(ctx, meta, index, s, numeric)
	return err
}

func (printer *MetricPrinter) printMetricRow(labels string, meta *MetricMeta, value string) error {
	var err error
	if !printer.headerPrinted {
		if meta.help != "" {
			fmt.Fprintf(printer.w, "# HELP %s %s\n", meta.snakeName, meta.help)
		}
		if meta.Type != "" {
			fmt.Fprintf(printer.w, "# TYPE %s %s\n", meta.snakeName, meta.Type)
		}
		printer.headerPrinted = true
	}
	if len(labels) == 0 {
		_, err = fmt.Fprintf(printer.w, "%s %s\n", meta.snakeName, value)
	} else {
		_, err = fmt.Fprintf(printer.w, "%s{%s} %s\n", meta.snakeName, labels, value)
	}
	return err
}

func (printer *MetricPrinter) Flush(ctx context.Context) error {

	labelMap := printer.metricBlock.LabelMap()

	output_count := 0
	for _, metricName := range printer.metricBlock.metricNames {
		values := printer.metricBlock.metrics[metricName]
		if values.Meta.IsLabel() {
			continue
		}
		printer.headerPrinted = false
		for _, row := range printer.metricBlock.rowIndexes {
			value, ok := values.Values[row]
			if ok {
				if !value.numeric {
					continue
				}
				labels := labelMap[row]
				err := printer.printMetricRow(labels, values.Meta, value.text)
				if err != nil {
					return err
				}
				output_count++
			}
		}
	}
	if output_count == 0 && printer.metricBlock.table != nil {
		tableMetricMeta := printer.metricBlock.table.metricMeta
		printer.headerPrinted = false
		for _, row := range printer.metricBlock.rowIndexes {
			labels := labelMap[row]
			err := printer.printMetricRow(labels, tableMetricMeta, "1")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (printer *MetricPrinter) calculateDisplayHint(object *mibdb.Object, meta *MetricMeta) {
	if meta.displayHint != "" {
		return
	}
	meta.displayHint, _ = object.Get("DISPLAY-HINT").(string)
	syntax := object.Get("SYNTAX")
findDisplayHint:
	for meta.displayHint == "" && syntax != nil {
		switch syntax2 := syntax.(type) {
		case *mibdb.TypeReference:
			meta.snmpType = syntax2.Name()
			switch meta.snmpType {
			case "INTEGER":
				meta.enums = syntax2.CompileEnums()
				if meta.enums == nil {
					meta.displayHint = "n"
				} else {
					meta.displayHint = "e"
					meta.flags |= MetricIsString
				}
				break findDisplayHint
			case "OBJECT IDENTIFIER":
				meta.displayHint = "a"
				meta.flags |= MetricIsString
			case "DisplayString":
				meta.displayHint = "a"
				meta.flags |= MetricIsString
			case "Opaque", "OCTET STRING":
				meta.displayHint = "b"
				meta.flags |= MetricIsString
			case "TimeTicks":
				meta.displayHint = "n"
			case "IpAddress", "NetworkAddress":
				meta.displayHint = "n."
				meta.flags |= MetricIsString
			case "PhysAddress":
				meta.displayHint = "x:"
				meta.flags |= MetricIsString
			case "Gauge32":
				meta.Type = "GAUGE"
				meta.displayHint = "n"
			case "Counter", "Counter32", "Counter64":
				meta.Type = "COUNTER"
				meta.displayHint = "n"
				if !strings.HasSuffix(meta.snakeName, "_total") {
					meta.snakeName += "_total"
				}
			default:
				def := printer.db.LookupName(syntax2.Name())
				syntax3, _ := def.(mibdb.Value)
				if syntax3 == syntax {
					break findDisplayHint
				}
				syntax = syntax3
			}
		case *mibdb.CompositeValue:
			syntax = syntax2.Get("SYNTAX")
			if syntax == syntax2 {
				break findDisplayHint
			}
		default:
			//println("syntax type =", reflect.TypeOf(syntax).String())
			break findDisplayHint
		}
	}
}

func (printer *MetricPrinter) ConvertCamelCaseToSnakeCase(name string) string {
	sb := strings.Builder{}
	var prev rune
	for _, c := range name {
		if (prev < 'A' || prev > 'Z') && c >= 'A' && c <= 'Z' {
			sb.WriteByte('_')
		}
		sb.WriteRune(c)
		prev = c
	}
	name = sb.String()
	name = strings.ToLower(name)
	return name
}

var spaceFmt = regexp.MustCompile(`\s+`)

func (printer *MetricPrinter) MetaDataForObject(object *mibdb.Object, children []*mibdb.Object) *MetricMeta {
	meta, _ := object.Get("metricMeta").(*MetricMeta)
	if meta != nil {
		return meta
	}

	name := object.Name()
	meta = &MetricMeta{
		name:      MetricName(name),
		snakeName: printer.ConvertCamelCaseToSnakeCase(name),
	}

	//if meta.Name == "tcpConnRemPort" || meta.Name == "tcpConnEntry" {
	//	println("debug", meta.SnakeName)
	//}

	meta.help, _ = object.Get("DESCRIPTION").(string)
	if meta.help != "" {
		meta.help = spaceFmt.ReplaceAllString(meta.help, " ")
		if len(meta.help) > 100 {
			dot := strings.Index(meta.help, ". ")
			if dot >= 0 {
				meta.help = meta.help[:dot+1]
			}
		}
		if meta.help != "" {
			meta.help += " "
		}
		meta.help += "(OID: " + object.OID().String() + ")"
	}

	printer.calculateDisplayHint(object, meta)
	index := object.Get("INDEX")
	if index != nil {
		meta.table = &Table{
			metricMeta: meta,
		}
		valueList, ok := index.(*mibdb.ValueList)
		if ok {
			for _, value := range *valueList {
				compositeValue, ok := value.(*mibdb.CompositeValue)
				if ok {
					v := compositeValue.Get("0")
					s, ok := v.(string)
					if ok {
						meta.table.index = append(meta.table.index, MetricName(s))
					}
				}
			}
		}
		for _, child := range children {
			metaChild := printer.MetaDataForObject(child, nil)
			if metaChild != nil {
				meta.table.columns = append(meta.table.columns, metaChild)
				if slices.Contains(meta.table.index, metaChild.name) {
					metaChild.flags |= MetricIsPartOfIndex
				}
				switch metaChild.displayHint {
				case "n":
					//ok
				case "e", "a":
					//meta.TableMeta.Index = append(meta.TableMeta.Index, metaChild.Name)
				case "":
					printer.calculateDisplayHint(child, metaChild)
					//println(metaChild.name, "display=[", metaChild.displayHint, "]")
				default:
					//println(metaChild.Name, "display=[", metaChild.DisplayHint, "]")
				}
			}
		}
		if meta.table.prefix == "" {
			if len(meta.table.columns) > 1 {
				meta.table.prefix = meta.table.columns[0].snakeName
				for i := 1; i < len(meta.table.columns); i++ {
					meta.table.prefix = findCommonPrefix(meta.table.prefix, meta.table.columns[i].snakeName)
				}
			}
		}
	}

	object.Set("metricMeta", meta)
	return meta
}

func (printer *MetricPrinter) MetaDataForBranch(branch *mibdb.OidBranch) *MetricMeta {
	object := branch.Object()
	meta, _ := object.Get("metricMeta").(*MetricMeta)
	if meta != nil {
		return meta
	}
	children := branch.ChildValues()
	meta = printer.MetaDataForObject(object, children)
	return meta
}

type unmarshallerFunc func(data asn1go.OID) (Value, asn1go.OID, error)

func (printer *MetricPrinter) Unmarshal(oidValue *mibdb.Object, data asn1go.OID) (Value, asn1go.OID, error) {
	var unmarshaller unmarshallerFunc = nil
	unmarshaller, _ = oidValue.Get("unmarshaller").(unmarshallerFunc)
	if unmarshaller == nil {
		//built it and store it in the oidValue
		//TODO really build it
		unmarshaller = func(data asn1go.OID) (Value, asn1go.OID, error) {
			if len(data) == 0 {
				return Value{}, data, asn1error.NewErrorf("OID element %d is truncated", 0)
			}
			head := data[0]
			data = data[1:]
			return Value{strconv.Itoa(head), true}, data, nil
		}
	}
	if unmarshaller != nil {
		return unmarshaller(data)
	}
	return Value{}, data, asn1error.NewUnimplementedError("MetricPrinter.Unmarshal not implemented")
}
