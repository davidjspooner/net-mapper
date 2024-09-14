package asn1go

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type OID []int

func (v *OID) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	b := bytes.Buffer{}

	if len(*v) < 2 {
		return asn1binary.Envelope{}, nil, asn1core.NewUnexpectedError(2, len(*v), "OID Prefix").WithUnits("elements")
	}
	b.WriteByte(byte((*v)[0]*40 + (*v)[1]))
	for i := 2; i < len(*v); i++ {
		n := (*v)[i]
		if n < 0 {
			return asn1binary.Envelope{}, nil, asn1core.NewErrorf("OID element %d is negative", n)
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
	return asn1binary.Envelope{Tag: asn1core.TagOID}, b.Bytes(), nil
}
func (v *OID) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1core.TagOID {
		return asn1core.NewUnexpectedError(asn1core.TagOID, envelope.Tag, "unexpected tag")
	}
	if len(bytes) < 1 {
		return asn1core.NewUnexpectedError(1, len(bytes), "OID Prefix").WithUnits("bytes")
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
			return asn1core.NewErrorf("OID element %d is truncated", n)
		}
		*v = append(*v, n)
	}
	return nil
}
func (o *OID) String() string {
	sb := strings.Builder{}
	for i, v := range *o {
		if i != 0 {
			sb.WriteString(".")
		}
		sb.WriteString(strconv.Itoa(v))
	}
	return sb.String()
}
