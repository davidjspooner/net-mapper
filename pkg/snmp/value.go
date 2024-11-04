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

type ValueFormatFunc func([]byte) any

var valueFormatFuncMap map[ValueType]ValueFormatFunc

func init() {
	valueFormatFuncMap = make(map[ValueType]ValueFormatFunc)
	valueFormatFuncMap[NullValue] = func(b []byte) any {
		return ""
	}
	valueFormatFuncMap[StringValue] = func(b []byte) any {
		v := asn1go.String{}
		v.UnpackAsn1(asn1binary.Envelope{Tag: asn1binary.TagOctetString}, b)
		return v.String()
	}
	valueFormatFuncMap[IntegerValue] = func(b []byte) any {
		v := asn1go.Integer{}
		v.UnpackAsn1(asn1binary.Envelope{Tag: asn1binary.TagInteger}, b)
		n, _ := v.GetInt(64)
		return n
	}
	valueFormatFuncMap[CounterValue] = valueFormatFuncMap[IntegerValue]
	valueFormatFuncMap[GaugeValue] = valueFormatFuncMap[IntegerValue]
	valueFormatFuncMap[TimeTicksValue] = valueFormatFuncMap[IntegerValue]
	valueFormatFuncMap[OidValue] = func(b []byte) any {
		v := asn1go.OID{}
		v.UnpackAsn1(asn1binary.Envelope{Tag: asn1binary.TagOID}, b)
		return v.String()
	}
	valueFormatFuncMap[IPValue] = func(b []byte) any {
		v := net.IP(b)
		return v.String()
	}
	valueFormatFuncMap[OpaqueValue] = func(b []byte) any {
		return b
	}
	valueFormatFuncMap[UnsignedValue] = valueFormatFuncMap[IntegerValue]
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
			s := branch.Object().Name()
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
