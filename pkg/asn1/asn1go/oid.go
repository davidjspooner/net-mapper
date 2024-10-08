package asn1go

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type OID []int

func (v *OID) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	b := bytes.Buffer{}

	if len(*v) < 2 {
		return asn1binary.Envelope{}, nil, asn1error.NewUnexpectedError(2, len(*v), "OID Prefix").WithUnits("elements")
	}
	b.WriteByte(byte((*v)[0]*40 + (*v)[1]))
	for i := 2; i < len(*v); i++ {
		n := (*v)[i]
		if n < 0 {
			return asn1binary.Envelope{}, nil, asn1error.NewErrorf("OID element %d is negative", n)
		}
		if n < 128 {
			b.WriteByte(byte(n))
		} else {
			var reverse [10]byte
			j := 0
			for n > 0 {
				reverse[j] = byte(n & 0x7F)
				n >>= 7
				j++
			}
			for j--; j >= 0; j-- {
				if j > 0 {
					b.WriteByte(reverse[j] | 0x80)
				} else {
					b.WriteByte(reverse[j])
				}
			}
		}
	}
	return asn1binary.Envelope{Tag: asn1binary.TagOID}, b.Bytes(), nil
}
func (v *OID) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1binary.TagOID {
		return asn1error.NewUnexpectedError(asn1binary.TagOID, envelope.Tag, "unexpected tag")
	}
	if len(bytes) < 1 {
		return asn1error.NewUnexpectedError(1, len(bytes), "OID Prefix").WithUnits("bytes")
	}
	*v = make(OID, 0, 10)
	*v = append(*v, int(bytes[0]/40), int(bytes[0]%40))
	for i := 1; i < len(bytes); {
		n := 0
		endOfOID := false
		for ; i < len(bytes); i++ {
			n = n<<7 + int(bytes[i]&0x7F)
			if bytes[i]&0x80 == 0 {
				endOfOID = true
				i++
				break
			}
		}
		if !endOfOID {
			return asn1error.NewErrorf("OID element %d is truncated", n)
		}
		*v = append(*v, n)
	}
	return nil
}
func (o OID) String() string {
	sb := strings.Builder{}
	for i, v := range o {
		if i != 0 {
			sb.WriteString(".")
		}
		sb.WriteString(strconv.Itoa(v))
	}
	return sb.String()
}

func ParseOID(s string, oidLookupFn func(string) (OID, error)) (OID, error) {
	parts := strings.Split(s, ".")
	oid := make(OID, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, asn1error.NewErrorf("OID element %d of %q is empty", i, s)
		}
		if part[0] >= '0' && part[0] <= '9' {
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, asn1error.NewErrorf("OID element %d of %q is not a number", i, s)
			}
			oid = append(oid, n)
		} else if oidLookupFn != nil {
			lookup, err := oidLookupFn(part)
			if err != nil {
				return nil, err
			}
			oid = append(oid, lookup...)
		} else {
			return nil, asn1error.NewErrorf("unable to resolve OID element %d of %q", i, s)
		}
	}
	return oid, nil
}
