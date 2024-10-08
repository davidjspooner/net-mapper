package snmp

import (
	"fmt"
	"net"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibdb"
)

type ValueType int

const (
	NullValue ValueType = iota
	StringValue
	CounterValue
	GaugeValue
	TimeTicksValue
	OidValue
	IntegerValue
	UnsignedValue
	IPValue
	OpaqueValue
)

func (t ValueType) String() string {
	switch t {
	case NullValue:
		return "Null"
	case StringValue:
		return "String"
	case CounterValue:
		return "Counter"
	case GaugeValue:
		return "Gauge"
	case TimeTicksValue:
		return "TimeTicks"
	case OidValue:
		return "OID"
	case IntegerValue:
		return "Integer"
	case UnsignedValue:
		return "Unsigned"
	case IPValue:
		return "IP"
	case OpaqueValue:
		return "Opaque"
	}
	return "Unknown"
}

func DecodeValue(db *mibdb.Database, v *asn1binary.Value) (string, ValueType, error) {
	switch v.Class {
	case asn1binary.ClassUniversal:
		switch v.Tag {
		case asn1binary.TagNull:
			return "", NullValue, nil
		case asn1binary.TagOctetString:
			var s asn1go.String
			err := v.UnpackIntoGo(&s)
			return s.String(), StringValue, err
		case asn1binary.TagOID:
			var oid asn1go.OID
			err := v.UnpackIntoGo(&oid)
			// Look up the OID in the MIB database
			branch, tail := db.FindOID(oid)
			s := branch.Name()
			if len(tail) > 0 {
				s += "." + tail.String()
			}
			return s, OidValue, err
		case asn1binary.TagInteger:
			var n asn1go.Integer
			err := v.UnpackIntoGo(&n)
			return n.String(), IntegerValue, err
		}
	case asn1binary.ClassApplication:
		switch v.Tag {
		case 0:
			var ip net.IP = v.Bytes
			return ip.String(), IPValue, nil
		case 1, 6:
			var n asn1go.Integer
			err := v.UnpackIntoGo(&n)
			return n.String(), CounterValue, err
		case 2:
			var n asn1go.Integer
			err := v.UnpackIntoGo(&n)
			return n.String(), GaugeValue, err
		case 3:
			var n asn1go.Integer
			err := v.UnpackIntoGo(&n)
			return n.String(), TimeTicksValue, err
		}
	}
	return "", NullValue, fmt.Errorf("unsupported value type %v", v)
}
