package snmp

import (
	"fmt"
	"io"

	"github.com/davidjspooner/net-mapper/pkg/snmp/mibdb"
)

type VarBindHandler interface {
	Handle(VarBind *VarBind) error
	Flush() error
}

//-------------------------------------

type VarBindHandlerFunc func(VarBind *VarBind) error

func (f VarBindHandlerFunc) Handle(vb *VarBind) error {
	return f(vb)
}
func (f VarBindHandlerFunc) Flush() error {
	return f(nil)
}

//-------------------------------------

type MetricPrinter struct {
	w  io.Writer
	db *mibdb.Database
}

var _ VarBindHandler = &MetricPrinter{}

func NewMetricPrinter(w io.Writer, db *mibdb.Database) *MetricPrinter {
	return &MetricPrinter{w: w, db: db}
}

func (printer *MetricPrinter) Handle(vb *VarBind) error {
	branch, tail := printer.db.FindOID(vb.OID)
	if branch == nil || branch.Definition() == nil {
		fmt.Fprintf(printer.w, "             OID: %s Value: %v\n", vb.OID.String(), vb.Value)
		return nil
	}
	name := branch.Name()
	parent := branch.Parent()
	parentDef := parent.Definition()
	_ = parentDef
	if len(tail) == 1 && tail[0] == 0 {
		fmt.Fprintf(printer.w, "             OID: %s Value: %v\n", name, vb.Value)
		return nil
	}
	fmt.Fprintf(printer.w, "             OID: %s.%s Value: %v\n", name, tail.String(), vb.Value)
	return nil
}

func (m *MetricPrinter) Flush() error {
	return nil
}
