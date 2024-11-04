package snmp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

	metricBlock MetricBlock
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

	if metaParent.TableMeta == nil && len(index) > 1 {
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
	if strings.Contains(meta.DisplayHint, "x:") {
		sb := strings.Builder{}
		for i, b := range []byte(s) {
			if i > 0 {
				sb.WriteString(":")
			}
			sb.WriteString(fmt.Sprintf("%02x", b))
		}
		s = sb.String()
	} else if meta.DisplayHint == "e" {
		n, _ := strconv.Atoi(s)
		var ok bool
		s, ok = meta.Enums[n]
		if !ok {
			s = fmt.Sprintf("%d", n)
		}
	} else if !strings.Contains(meta.DisplayHint, "a") {
		err = printer.queueLine(ctx, meta, metaParent.TableMeta, index, Value{s, true})
		return err
	}
	err = printer.queueLine(ctx, meta, metaParent.TableMeta, index, Value{s, false})
	return err
}

func (printer *MetricPrinter) queueLine(ctx context.Context, meta *MetricMeta, table *TableMeta, index asn1go.OID, value Value) error {
	if (printer.lastPrintedMeta != meta) && (printer.metricBlock.TableMeta != table || table == nil) {
		err := printer.Flush(ctx)
		if err != nil {
			return err
		}
		printer.metricBlock.Init(table)
		printer.lastPrintedMeta = meta
	}

	row := RowIndex(index.String())

	if printer.metricBlock.IsNewRow(row) && printer.metricBlock.TableMeta != nil {
		tail := index
		var err error
		var value2 Value
		for _, columnName := range printer.metricBlock.TableMeta.Index {
			def, _ := printer.db.LookupName(string(columnName))
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
func (printer *MetricPrinter) Flush(ctx context.Context) error {

	labelMap := printer.metricBlock.LabelMap()

	for _, metricName := range printer.metricBlock.MetricNames {

		if printer.metricBlock.TableMeta != nil {
			if slices.Contains(printer.metricBlock.TableMeta.Index, metricName) {
				continue
			}
		}

		values := printer.metricBlock.Metrics[metricName]
		if values.Meta.Help != "" {
			fmt.Fprintf(printer.w, "# HELP %s %s\n", values.Meta.Name, values.Meta.Help)
		}
		if values.Meta.Type != "" {
			fmt.Fprintf(printer.w, "# TYPE %s %s\n", values.Meta.Name, values.Meta.Type)
		}
		for _, row := range printer.metricBlock.RowIndexes {
			value, ok := values.Values[row]
			if ok {
				var err error
				labels := labelMap[row]
				if value.Numeric {
					if len(labels) == 0 {
						_, err = fmt.Fprintf(printer.w, "%s %s\n", values.Meta.Name, value.Text)
					} else {
						_, err = fmt.Fprintf(printer.w, "%s{%s} %s\n", values.Meta.Name, labels, value.Text)
					}
				} else {
					if len(labels) == 0 {
						_, err = fmt.Fprintf(printer.w, "%s{value=%q} 1\n", values.Meta.Name, value.Text)
					} else {
						_, err = fmt.Fprintf(printer.w, "%s{%s,value=%q} 1\n", values.Meta.Name, labels, value.Text)
					}
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (printer *MetricPrinter) MetaDataForObject(object *mibdb.Object, children []*mibdb.Object) *MetricMeta {
	meta, _ := object.Get("metricMeta").(*MetricMeta)
	if meta != nil {
		return meta
	}

	meta = &MetricMeta{
		Name: MetricName(object.Name()),
	}
	meta.Help, _ = object.Get("DESCRIPTION").(string)
	if meta.Help != "" {
		if len(meta.Help) > 100 {
			dot := strings.IndexByte(meta.Help, '.')
			if dot >= 0 {
				meta.Help = meta.Help[:dot+1]
			}
		}
		if meta.Help != "" {
			meta.Help += " "
		}
		meta.Help += "(OID: " + object.OID().String() + ")"
	}

	meta.DisplayHint, _ = object.Get("DISPLAY-HINT").(string)
	syntax := object.Get("SYNTAX")
findDisplayHint:
	for meta.DisplayHint == "" && syntax != nil {
		switch syntax2 := syntax.(type) {
		case *mibdb.TypeReference:
			meta.SnmpType = syntax2.Name()
			switch meta.SnmpType {
			case "INTEGER":
				meta.Enums = syntax2.CompileEnums()
				if meta.Enums == nil {
					meta.DisplayHint = "n"
				} else {
					meta.DisplayHint = "e"
				}
				break findDisplayHint
			case "OBJECT IDENTIFIER":
				meta.DisplayHint = "a"
			case "DisplayString":
				meta.DisplayHint = "a"
			case "Opaque":
				meta.DisplayHint = "b"
			case "TimeTicks":
				meta.DisplayHint = "n"
			case "IpAddress", "NetworkAddress":
				meta.DisplayHint = "n."
			case "PhysAddress":
				meta.DisplayHint = "x:"
			case "Gauge32":
				meta.Type = "GAUGE"
				meta.DisplayHint = "n"
			case "Counter", "Counter32", "Counter64":
				meta.Type = "COUNTER"
				meta.DisplayHint = "n"
			default:
				def, _ := printer.db.LookupName(syntax2.Name())
				syntax3, _ := def.(mibdb.Value)
				if syntax3 == syntax {
					break findDisplayHint
				}
				syntax = syntax3
			}
		default:
			break findDisplayHint
		}
	}
	index := object.Get("INDEX")
	if index != nil {
		meta.TableMeta = &TableMeta{}
		valueList, ok := index.(*mibdb.ValueList)
		if ok {
			for _, value := range *valueList {
				compositeValue, ok := value.(*mibdb.CompositeValue)
				if ok {
					v := compositeValue.Get("0")
					s, ok := v.(string)
					if ok {
						meta.TableMeta.Index = append(meta.TableMeta.Index, MetricName(s))
					}
				}
			}
		}
	}
	if meta.TableMeta != nil && len(children) > 0 && len(meta.TableMeta.Columns) == 0 {
		for _, child := range children {
			metaChild := printer.MetaDataForObject(child, nil)
			if metaChild != nil {
				meta.TableMeta.Columns = append(meta.TableMeta.Columns, metaChild)
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
