package snmp

import (
	"context"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
)

const (
	GET      = asn1binary.Tag(0x20)
	GET_NEXT = asn1binary.Tag(0x21)
	GET_BULK = asn1binary.Tag(0x25)
	SET      = asn1binary.Tag(0x23)
	TRAP     = asn1binary.Tag(0x24)
	INFORM   = asn1binary.Tag(0x26)
	RESPONSE = asn1binary.Tag(0x22)
)

type Connection interface {
	Send(pType asn1binary.Tag, pdu *PDU) error
	Receive() (*PDU, error)
	Close() error
}

type Protocol interface {
	Dial(target string) (Connection, error)
	DecodeFrame(frame []byte) (*Message, error)
	EncodePDU(pType asn1binary.Tag, pdu *PDU) ([]byte, error)
}

type VarBind struct {
	OID   asn1go.OID
	Value asn1binary.Value
}

func (vb *VarBind) String() string {
	return vb.OID.String() + ": " + vb.Value.String()
}

type PDU struct {
	asn1binary.Envelope
	RequestID   int
	ErrorStatus int
	ErrorIndex  int
	VarBinds    []VarBind `asn1:"Sequence,Constructed"`
}

type Message struct {
	Version   int
	Community string `asn1:"OctetString"`
	PDU       PDU
}

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
