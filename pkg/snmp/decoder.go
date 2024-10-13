package snmp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibdb"
)

type VarBindHandler interface {
	Handle(ctx context.Context, VarBind *VarBind) error
	Flush(ctx context.Context) error
}

//-------------------------------------

type VarBindHandlerFunc func(ctx context.Context, VarBind *VarBind) error

func (f VarBindHandlerFunc) Handle(ctx context.Context, vb *VarBind) error {
	return f(ctx, vb)
}
func (f VarBindHandlerFunc) Flush(ctx context.Context) error {
	return f(ctx, nil)
}

//-------------------------------------

type TableStructure struct {
	Columns []string
}

type MetricMeta struct {
	Name             string
	Help, Type, Unit string
	DisplayHint      string
	Enums            map[int]string
	Table            *TableStructure
}

type MetricPrinter struct {
	w               io.Writer
	db              *mibdb.Database
	lastPrintedMeta *MetricMeta
}

var _ VarBindHandler = &MetricPrinter{}

func NewMetricPrinter(w io.Writer, db *mibdb.Database) *MetricPrinter {
	return &MetricPrinter{w: w, db: db}
}

func (printer *MetricPrinter) Handle(ctx context.Context, vb *VarBind) error {

	s, _, err := DecodeValue(printer.db, &vb.Value)
	if err != nil {
		printer.db.Logger().WarnContext(ctx, "Error decoding value", slog.String("OID", vb.OID.String()), slog.String("error", err.Error()))
		return err
	}
	if len(vb.OID) > 0 && vb.OID[0] == 0 {
		vb.OID = vb.OID[1:]
	}

	branch, tail := printer.db.FindOID(vb.OID)
	if branch == nil || branch.Value() == nil {
		printer.db.Logger().WarnContext(ctx, "OID not found", slog.String("OID", vb.OID.String()))
		return nil
	}

	meta := printer.MetaDataForBranch(branch)
	metaParent := printer.MetaDataForBranch(branch.Parent())
	_ = metaParent

	if metaParent.Table == nil && len(tail) > 1 {
		var oid asn1go.OID
		oid = append(oid, branch.Value().OID()...)
		oid = append(oid, tail[0])
		printer.db.Logger().WarnContext(ctx, "Oid not known", slog.String("OID", oid.String()+".*"))
		return nil
	}

	printer.printLine(ctx, meta, tail, s)
	return nil
}

func (printer *MetricPrinter) printLine(_ context.Context, meta *MetricMeta, _ asn1go.OID, s string) error {
	if printer.lastPrintedMeta != meta {
		printer.lastPrintedMeta = meta
		if meta.Help != "" {
			fmt.Fprintf(printer.w, "# HELP %s %s\n", meta.Name, meta.Help)
		}
		if meta.Type != "" {
			fmt.Fprintf(printer.w, "# TYPE %s %s\n", meta.Name, meta.Type)
		}
		if meta.Unit != "" {
			fmt.Fprintf(printer.w, "# UNIT %s %s\n", meta.Name, meta.Unit)
		}
	}
	var err error
	if strings.Contains(meta.DisplayHint, "x:") {
		sb := strings.Builder{}
		for i, b := range []byte(s) {
			if i > 0 {
				sb.WriteString(":")
			}
			sb.WriteString(fmt.Sprintf("%02x", b))
		}
		s = sb.String()
		_, err = fmt.Fprintf(printer.w, "%s{value=%q} 1\n", meta.Name, s)
	} else if strings.Contains(meta.DisplayHint, "a") {
		_, err = fmt.Fprintf(printer.w, "%s{value=%q} 1\n", meta.Name, s)
	} else if meta.DisplayHint == "e" {
		n, _ := strconv.Atoi(s)
		s, ok := meta.Enums[n]
		if !ok {
			s = fmt.Sprintf("%d", n)
		}
		_, err = fmt.Fprintf(printer.w, "%s{value=\"%s\"} 1\n", meta.Name, s)
	} else {
		_, err = fmt.Fprintf(printer.w, "%s %s\n", meta.Name, s)
	}
	return err
}

func (printer *MetricPrinter) Flush(ctx context.Context) error {
	return nil
}

func (printer *MetricPrinter) MetaDataForBranch(branch *mibdb.OidBranch) *MetricMeta {
	meta, _ := branch.Get("metricMeta").(*MetricMeta)
	if meta != nil {
		return meta
	}

	//generate meta
	meta = &MetricMeta{
		Name: branch.Value().Name(),
	}
	value := branch.Value()

	if meta.Name == "ifType" {
		print("debug - ifType")
	}

	meta.Help, _ = value.Get("DESCRIPTION").(string)
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
		meta.Help += "(OID: " + branch.Value().OID().String() + ")"
	}

	meta.DisplayHint, _ = value.Get("DISPLAY-HINT").(string)
	syntax := value.Get("SYNTAX")
findDisplayHint:
	for meta.DisplayHint == "" && syntax != nil {
		switch syntax2 := syntax.(type) {
		case *mibdb.TypeReference:
			//TODO better lookup of type and thus inferring of DisplayHint
			switch syntax2.Name() {
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
			case "IpAddress", "NetworkAddress":
				meta.DisplayHint = "a"
			case "PhysAddress":
				meta.DisplayHint = "x:"
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

	index := value.Get("INDEX")
	if index != nil {
		meta.Table = &TableStructure{}
		valueList, ok := index.(*mibdb.ValueList)
		if ok {
			for _, value := range *valueList {
				compositeValue, ok := value.(*mibdb.CompositeValue)
				if ok {
					v := compositeValue.Get("0")
					s, ok := v.(string)
					if ok {
						meta.Table.Columns = append(meta.Table.Columns, s)
					}
				}
			}
		}
		print(strings.Join(meta.Table.Columns, ",") + "\n")
	}

	branch.Set("metricMeta", meta)
	return meta
}
