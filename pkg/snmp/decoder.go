package snmp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"

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

type MetricPrinter struct {
	w          io.Writer
	db         *mibdb.Database
	lastBranch *mibdb.OidBranch
}

var _ VarBindHandler = &MetricPrinter{}

func NewMetricPrinter(w io.Writer, db *mibdb.Database) *MetricPrinter {
	return &MetricPrinter{w: w, db: db}
}

func (printer *MetricPrinter) Handle(ctx context.Context, vb *VarBind) error {

	s, valueType, err := DecodeValue(printer.db, &vb.Value)
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

	printer.print(ctx, branch, tail, valueType, s)

	return nil
}

func (printer *MetricPrinter) print(_ context.Context, branch *mibdb.OidBranch, tail asn1go.OID, valueType ValueType, s string) error {
	name := branch.Name()
	if len(tail) > 0 {
		if len(tail) != 1 || tail[0] != 0 {
			for _, i := range tail {
				name += fmt.Sprintf("_%d", i)
			}
		}
	}

	index := ""
	metricType := "unknown"
	unit := ""
	switch valueType {
	case CounterValue:
		metricType = "counter"
	case NullValue:
		return nil
	case GaugeValue:
		metricType = "gauge"
	case UnsignedValue, IntegerValue:
		metricType = "gauge"
	case TimeTicksValue:
		f, _ := strconv.ParseFloat(s, 64)
		metricType = "counter"
		unit = "seconds"
		s = strconv.FormatFloat(f/100, 'f', -1, 64)
	default:
		index = s
		s = "1"
		metricType = "info"
	}
	if printer.lastBranch != branch {
		fmt.Fprintf(printer.w, "# TYPE %s %s\n", name, metricType)
		if unit != "" {
			fmt.Fprintf(printer.w, "# UNIT %s %s\n", name, unit)
		}
	}

	if index == "" {
		fmt.Fprintf(printer.w, "%s %s\n", name, s)
		return nil
	}
	fmt.Fprintf(printer.w, "%s{value=%q} %s\n", name, index, s)
	return nil
}

func (m *MetricPrinter) Flush(ctx context.Context) error {
	return nil
}
