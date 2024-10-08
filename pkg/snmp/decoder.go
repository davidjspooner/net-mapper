package snmp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

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

type MetricMeta struct {
	Help, Type, Unit string
	DisplayHint      string
}

type MetricPrinter struct {
	w        io.Writer
	db       *mibdb.Database
	lastMeta *MetricMeta
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
	if branch == nil || branch.Definition() == nil {
		printer.db.Logger().WarnContext(ctx, "OID not found", slog.String("OID", vb.OID.String()))
		return nil
	}
	parent := branch.Parent()
	parentDef := parent.Definition()
	_ = parentDef

	name := branch.Name()
	if len(tail) > 0 {
		if len(tail) != 1 || tail[0] != 0 {
			for _, i := range tail {
				name += fmt.Sprintf("_%d", i)
			}
		}
	}

	meta := printer.MetaDataForBranch(branch)
	if printer.lastMeta != meta {
		printer.lastMeta = meta
		if meta.Help != "" {
			fmt.Fprintf(printer.w, "# HELP %s %s\n", name, meta.Help)
		}
		if meta.Type != "" {
			fmt.Fprintf(printer.w, "# TYPE %s %s\n", name, meta.Type)
		}
		if meta.Unit != "" {
			fmt.Fprintf(printer.w, "# UNIT %s %s\n", name, meta.Unit)
		}
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
		fmt.Fprintf(printer.w, "%s{value=%q} 1\n", name, s)
	} else if strings.Contains(meta.DisplayHint, "a") {
		fmt.Fprintf(printer.w, "%s{value=%q} 1\n", name, s)
	} else {
		fmt.Fprintf(printer.w, "%s %s\n", name, s)
	}
	return nil
}

func (m *MetricPrinter) Flush(ctx context.Context) error {
	return nil
}

func (printer *MetricPrinter) MetaDataForBranch(branch *mibdb.OidBranch) *MetricMeta {
	meta, _ := branch.Get("metricMeta").(*MetricMeta)
	if meta != nil {
		return meta
	}

	//generate meta
	meta = &MetricMeta{}
	value, ok := branch.Definition().(mibdb.Value)
	if !ok {
		return meta
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
		meta.Help += "(OID: " + branch.OID().String() + ")"
	}
	meta.DisplayHint, _ = value.Get("DISPLAY-HINT").(string)
	if meta.DisplayHint == "" {
		syntax := value.Get("SYNTAX")
		if syntax != nil {
			switch syntax := syntax.(type) {
			case *mibdb.TypeReference:
				syntaxType := syntax.Lookup()
				switch syntaxType := syntaxType.(type) {
				case *mibdb.CompositeValue:
					meta.DisplayHint, _ = syntaxType.Get("DISPLAY-HINT").(string)
				default:
					//pass
				}
				switch syntax.Name() {
				case "IpAddress", "NetworkAddress":
					meta.DisplayHint = "a"
				case "PhysAddress":
					meta.DisplayHint = "x:"
				case "Counter", "Counter32", "Counter64":
					meta.Type = "COUNTER"
				}
			default:
				//pass
			}
		}
	}
	branch.Set("metricMeta", meta)
	return meta
}
