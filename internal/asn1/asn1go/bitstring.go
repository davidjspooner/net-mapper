package asn1go

import (
	"encoding/hex"
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type BitString struct {
	Unused byte
	Bytes  []byte
}

func (v *BitString) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	b := make([]byte, 1, len(v.Bytes)+1)
	b[0] = v.Unused
	b = append(b, v.Bytes...)
	return asn1binary.Envelope{Tag: asn1core.TagBitString}, b, nil
}
func (v *BitString) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1core.TagBitString {
		return asn1core.NewUnexpectedError(asn1core.TagBitString, envelope.Tag, "unexpected tag")
	}
	if len(bytes) < 1 {
		return asn1core.NewUnexpectedError(1, len(bytes), "bitstring prefix").WithUnits("bytes")
	}
	v.Unused = bytes[0]
	v.Bytes = make([]byte, len(bytes)-1)
	copy(v.Bytes, bytes[1:])
	return nil
}
func (v *BitString) String() string {
	return fmt.Sprintf("%s (%d unused)", hex.EncodeToString(v.Bytes), v.Unused)
}
