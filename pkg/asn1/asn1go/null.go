package asn1go

import (
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
)

//--------------------------------------------------------------------------------------------

type Null struct {
}

func (v *Null) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{Tag: asn1core.TagNull}, nil, nil
}
func (v *Null) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1core.TagNull {
		return asn1core.NewUnexpectedError(asn1core.TagNull, envelope.Tag, "unexpected tag")
	}
	if len(bytes) != 0 {
		return asn1core.NewUnexpectedError(0, len(bytes), "null").WithUnits("bytes")
	}
	return nil
}
func (v *Null) String() string {
	return "null"
}
